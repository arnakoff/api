// SPDX-License-Identifier: AGPL-3.0-only

// Package backupseal encrypts the backups of the key store with age
// (filippo.io/age), for the public key of the owner (#75): the process
// that seals a backup cannot open it. It also opens a dump sealed for a
// key drawn for one restore in production (#297).
package backupseal

import (
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
}

// New returns a Sealer for recipient, one native age public key, such as
// the "age1pq1…" printed by age-keygen -pq, or ErrRecipient.
func New(recipient string) (*Sealer, error) {
	rs, err := age.ParseRecipients(strings.NewReader(recipient))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRecipient, err)
	}
	if len(rs) != 1 {
		return nil, fmt.Errorf("%w: %d keys", ErrRecipient, len(rs))
	}
	return &Sealer{recipient: rs[0]}, nil
}

// Seal returns a WriteCloser encrypting what is written to it into w; Close
// writes the last chunk.
func (s *Sealer) Seal(w io.Writer) (io.WriteCloser, error) {
	wc, err := age.Encrypt(w, s.recipient)
	if err != nil {
		return nil, fmt.Errorf("age: %w", err)
	}
	return wc, nil
}
