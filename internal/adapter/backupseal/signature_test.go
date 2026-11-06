// SPDX-License-Identifier: AGPL-3.0-only

package backupseal_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"filippo.io/age"

	"github.com/arnakoff/api/internal/adapter/backupseal"
	"github.com/arnakoff/api/internal/app"
)

// A throwaway test pair, drawn with the commands of docs/deploy.md:
// openssl genpkey -algorithm ed25519 -outform DER, then the last 32 bytes
// of the private key and of its public key, in hex.
const (
	testSigningKey = "233b01916151fe156d809150c20bb942c8692d2b76eec18ba46e77863ac2c3e5"
	testVerifyKey  = "07b769cae872f43683b42b114b802a78e5c1cd1b5a58a6e3cf4e9868605e4873"
	otherVerifyKey = "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c"
)

var _ app.KeyDumpOpener = (*backupseal.Verifier)(nil)

// sealSigned returns plain sealed for id and signed with signingKey.
func sealSigned(t *testing.T, id *age.HybridIdentity, signingKey, plain string) []byte {
	t.Helper()
	signer, err := backupseal.NewSigner(signingKey)
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}
	s, err := backupseal.New(id.Recipient().String(), signer)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	var sealed bytes.Buffer
	w, err := s.Seal(&sealed)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if _, err := io.WriteString(w, plain); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return sealed.Bytes()
}

func testSigner(t *testing.T) *backupseal.Signer {
	t.Helper()
	s, err := backupseal.NewSigner(testSigningKey)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func testVerifier(t *testing.T) *backupseal.Verifier {
	t.Helper()
	v, err := backupseal.NewVerifier(testVerifyKey)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func newIdentity(t *testing.T) *age.HybridIdentity {
	t.Helper()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func opener(t *testing.T, id *age.HybridIdentity, verifyKey string) *backupseal.Opener {
	t.Helper()
	v, err := backupseal.NewVerifier(verifyKey)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	o, err := backupseal.NewOpener(id.String(), v)
	if err != nil {
		t.Fatalf("NewOpener() error = %v", err)
	}
	return o
}

func TestSignedDumpOpensWithTheVerifyKey(t *testing.T) {
	t.Parallel()
	id := newIdentity(t)
	const plain = "manifest.json and api_keys.csv"
	got, err := opener(t, id, testVerifyKey).OpenDump(sealSigned(t, id, testSigningKey, plain))
	if err != nil {
		t.Fatalf("OpenDump() error = %v", err)
	}
	if string(got) != plain {
		t.Errorf("OpenDump() = %q, want %q: the signature is not part of the dump", got, plain)
	}
}

func TestOpenDumpRefusesADumpSignedByAnotherKey(t *testing.T) {
	t.Parallel()
	id := newIdentity(t)
	sealed := sealSigned(t, id, testSigningKey, "dump")
	if _, err := opener(t, id, otherVerifyKey).OpenDump(sealed); !errors.Is(err, backupseal.ErrSignature) {
		t.Errorf("OpenDump() error = %v, want ErrSignature", err)
	}
}

func TestOpenDumpRefusesAnUnsignedDump(t *testing.T) {
	t.Parallel()
	// Anyone holding the public recipient can seal a dump (#340, point 3).
	id := newIdentity(t)
	var sealed bytes.Buffer
	w, err := age.Encrypt(&sealed, id.Recipient())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "forged dump with no signature, long enough to hold a trailer........................................"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := opener(t, id, testVerifyKey).OpenDump(sealed.Bytes()); !errors.Is(err, backupseal.ErrSignature) {
		t.Errorf("OpenDump(unsigned) error = %v, want ErrSignature", err)
	}
}

func TestVerifierChecksAClearDump(t *testing.T) {
	t.Parallel()
	// admin restore -file loads a dump the owner decrypted: the signature
	// travels with it.
	id := newIdentity(t)
	r, err := age.Decrypt(bytes.NewReader(sealSigned(t, id, testSigningKey, "dump")), id)
	if err != nil {
		t.Fatal(err)
	}
	clearDump, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	v, err := backupseal.NewVerifier(testVerifyKey)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := v.OpenDump(clearDump); err != nil || string(got) != "dump" {
		t.Errorf("OpenDump(clear) = %q, %v; want \"dump\"", got, err)
	}
	clearDump[0] ^= 1
	if _, err := v.OpenDump(clearDump); !errors.Is(err, backupseal.ErrSignature) {
		t.Errorf("OpenDump(altered) error = %v, want ErrSignature", err)
	}
}

func TestKeysMustBe32BytesOfHex(t *testing.T) {
	t.Parallel()
	for _, k := range []string{"", "zz", testSigningKey[:62], testSigningKey + "00"} {
		if _, err := backupseal.NewSigner(k); !errors.Is(err, backupseal.ErrSigningKey) {
			t.Errorf("NewSigner(%d chars) error = %v, want ErrSigningKey", len(k), err)
		}
		if _, err := backupseal.NewVerifier(k); !errors.Is(err, backupseal.ErrVerifyKey) {
			t.Errorf("NewVerifier(%d chars) error = %v, want ErrVerifyKey", len(k), err)
		}
	}
	if _, err := backupseal.New(newIdentity(t).Recipient().String(), nil); err == nil {
		t.Error("New(recipient, nil signer) error = nil, want an error: every backup is signed")
	}
}
