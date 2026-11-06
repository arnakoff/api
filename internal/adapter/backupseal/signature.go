// SPDX-License-Identifier: AGPL-3.0-only

package backupseal

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
)

// age seals for a public recipient and authenticates nothing about the
// sender: anyone holding the recipient could seal a dump. The backup job
// therefore signs each dump with a key only it holds, and a restore
// refuses a dump that the matching public key does not verify (#340,
// point 3). The clear dump is the tar, then signatureMarker, then the
// Ed25519ph signature of the tar, all sealed together.

// signatureMarker separates the tar from its signature.
const signatureMarker = "\narnakoff-dump-signature-v1\n"

// signatureContext binds the signature to this use of the key.
const signatureContext = "arnakoff key-store dump"

var (
	// ErrSigningKey means the signing key is not 32 bytes of hex, the seed
	// of an Ed25519 private key.
	ErrSigningKey = errors.New("backup signing key must be 32 bytes of hex")

	// ErrVerifyKey means the verify key is not 32 bytes of hex, an Ed25519
	// public key.
	ErrVerifyKey = errors.New("restore verify key must be 32 bytes of hex")

	// ErrSignature means the dump carries no signature, or one the verify
	// key does not accept: it was not made by the backup job.
	ErrSignature = errors.New("dump signature missing or invalid")
)

// signOptions are the Ed25519ph options of every dump signature: the tar is
// hashed as it is written, so the clear dump is never held whole.
var signOptions = &ed25519.Options{Hash: crypto.SHA512, Context: signatureContext}

// Signer signs dumps with the private key of the backup job.
type Signer struct {
	key ed25519.PrivateKey
}

// NewSigner returns a Signer for seed, the hex of a 32-byte Ed25519 seed,
// or ErrSigningKey. The error never quotes seed.
func NewSigner(seed string) (*Signer, error) {
	b, err := hex32(seed)
	if err != nil {
		return nil, ErrSigningKey
	}
	return &Signer{key: ed25519.NewKeyFromSeed(b)}, nil
}

// Verifier checks the signature of a clear dump. It implements
// app.KeyDumpOpener for a dump the owner already decrypted.
type Verifier struct {
	key ed25519.PublicKey
}

// NewVerifier returns a Verifier for key, the hex of a 32-byte Ed25519
// public key, or ErrVerifyKey.
func NewVerifier(key string) (*Verifier, error) {
	b, err := hex32(key)
	if err != nil {
		return nil, ErrVerifyKey
	}
	return &Verifier{key: ed25519.PublicKey(b)}, nil
}

// OpenDump returns the tar of dump once its signature is verified, or
// ErrSignature. The tar shares dump's memory.
func (v *Verifier) OpenDump(dump []byte) ([]byte, error) {
	cut := len(dump) - ed25519.SignatureSize - len(signatureMarker)
	if cut < 0 || !bytes.Equal(dump[cut:cut+len(signatureMarker)], []byte(signatureMarker)) {
		return nil, ErrSignature
	}
	tar, sig := dump[:cut], dump[cut+len(signatureMarker):]
	digest := sha512.Sum512(tar)
	if err := ed25519.VerifyWithOptions(v.key, digest[:], sig, signOptions); err != nil {
		return nil, ErrSignature
	}
	return tar, nil
}

// signingWriter hashes what it passes on to the sealed writer, and on
// Close appends the marker and the signature before closing it.
type signingWriter struct {
	sealed io.WriteCloser
	digest hash.Hash
	signer *Signer
}

func (w *signingWriter) Write(p []byte) (int, error) {
	n, err := w.sealed.Write(p)
	w.digest.Write(p[:n])
	return n, err
}

// Close signs the dump written so far and closes the sealed writer.
func (w *signingWriter) Close() error {
	sig, err := w.signer.key.Sign(nil, w.digest.Sum(nil), signOptions)
	if err != nil {
		return fmt.Errorf("sign dump: %w", err)
	}
	if _, err := io.WriteString(w.sealed, signatureMarker); err != nil {
		return err
	}
	if _, err := w.sealed.Write(sig); err != nil {
		return err
	}
	return w.sealed.Close()
}

// hex32 decodes s, which must be exactly 32 bytes of hex.
func hex32(s string) ([]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return nil, errors.New("not 32 bytes of hex")
	}
	return b, nil
}
