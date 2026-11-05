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

var _ app.KeyDumpOpener = (*backupseal.Opener)(nil)

// seal returns plain encrypted for recipient.
func seal(t *testing.T, recipient age.Recipient, plain string) []byte {
	t.Helper()
	var sealed bytes.Buffer
	w, err := age.Encrypt(&sealed, recipient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, plain); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return sealed.Bytes()
}

func TestOpenerOpensWhatIsSealedForItsIdentity(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	o, err := backupseal.NewOpener(id.String())
	if err != nil {
		t.Fatalf("NewOpener() error = %v", err)
	}
	const plain = "jeanne.martin@example.org"
	got, err := o.OpenDump(seal(t, id.Recipient(), plain))
	if err != nil {
		t.Fatalf("OpenDump() error = %v", err)
	}
	if string(got) != plain {
		t.Errorf("OpenDump() = %q, want %q", got, plain)
	}
}

func TestOpenerRefusesADumpItCannotTrust(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	other, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	o, err := backupseal.NewOpener(id.String())
	if err != nil {
		t.Fatal(err)
	}
	sealed := seal(t, id.Recipient(), string(bytes.Repeat([]byte("x"), 100_000)))
	altered := bytes.Clone(sealed)
	altered[len(altered)-1] ^= 1
	tests := map[string][]byte{
		"empty":                  nil,
		"not age":                []byte("keys.tar"),
		"for another identity":   seal(t, other.Recipient(), "dump"),
		"cut short":              sealed[:len(sealed)-1000],
		"altered in its last 1B": altered,
	}
	for name, dump := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got, err := o.OpenDump(dump); err == nil {
				t.Errorf("OpenDump() = %d bytes, want an error", len(got))
			}
		})
	}
}

func TestNewOpenerRefuses(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"empty":          "",
		"comments only":  "# no key\n",
		"a public key":   id.Recipient().String(),
		"two identities": id.String() + "\n" + id.String(),
	}
	for name, identity := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := backupseal.NewOpener(identity); !errors.Is(err, backupseal.ErrIdentity) {
				t.Errorf("NewOpener() error = %v, want ErrIdentity", err)
			}
		})
	}
}
