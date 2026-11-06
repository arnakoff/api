// SPDX-License-Identifier: AGPL-3.0-only

// Package backupseal encrypts the backups of the key store with age
// (filippo.io/age), for the public key of the owner (#75): the process
// that seals a backup cannot open it. It also opens a dump sealed for a
// key drawn for one restore in production (#297).
package backupseal

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// ErrRecipient means the recipient is not one native age public key.
var ErrRecipient = errors.New("backup recipient must be one age public key")

// Sealer encrypts backups for one recipient. It implements
// app.BackupSealer.
type Sealer struct {
	recipient age.Recipient
	signer    *Signer
}

// ErrNoSigner means a Sealer was asked for without a Signer: every backup
// is signed.
var ErrNoSigner = errors.New("backup sealer needs a signer")

// New returns a Sealer for recipient, one native age public key, such as
// the "age1pq1…" printed by age-keygen -pq, or ErrRecipient. Every dump it
// seals is signed by signer.
func New(recipient string, signer *Signer) (*Sealer, error) {
	if signer == nil {
		return nil, ErrNoSigner
	}
	rs, err := age.ParseRecipients(strings.NewReader(recipient))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRecipient, err)
	}
	if len(rs) != 1 {
		return nil, fmt.Errorf("%w: %d keys", ErrRecipient, len(rs))
	}
	return &Sealer{recipient: rs[0], signer: signer}, nil
}

// Seal returns a WriteCloser encrypting what is written to it into w; Close
// appends the signature of what was written, then writes the last chunk.
func (s *Sealer) Seal(w io.Writer) (io.WriteCloser, error) {
	wc, err := age.Encrypt(w, s.recipient)
	if err != nil {
		return nil, fmt.Errorf("age: %w", err)
	}
	return &signingWriter{sealed: wc, digest: sha512.New(), signer: s.signer}, nil
}
