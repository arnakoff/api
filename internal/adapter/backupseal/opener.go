// SPDX-License-Identifier: AGPL-3.0-only

package backupseal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// ErrIdentity means the identity is not one native age private key.
var ErrIdentity = errors.New("restore identity must be one age private key")

// Opener decrypts a dump sealed for one identity: in production, a key
// drawn for one restore, never the owner's (#297). It implements
// app.KeyDumpOpener.
type Opener struct {
	identity age.Identity
}

// NewOpener returns an Opener for identity, one native age private key,
// such as the "AGE-SECRET-KEY-PQ-1…" printed by age-keygen -pq, or
// ErrIdentity. The error never quotes identity.
func NewOpener(identity string) (*Opener, error) {
	ids, err := age.ParseIdentities(strings.NewReader(identity))
	if err != nil {
		return nil, ErrIdentity
	}
	if len(ids) != 1 {
		return nil, fmt.Errorf("%w: %d keys", ErrIdentity, len(ids))
	}
	return &Opener{identity: ids[0]}, nil
}

// OpenDump decrypts sealed whole: age authenticates every chunk and the
// end of the file, so a dump altered or cut short fails here.
func (o *Opener) OpenDump(sealed []byte) ([]byte, error) {
	r, err := age.Decrypt(bytes.NewReader(sealed), o.identity)
	if err != nil {
		return nil, fmt.Errorf("age: %w", err)
	}
	dump, err := io.ReadAll(r)
	if err != nil {
		clear(dump)
		return nil, fmt.Errorf("age: %w", err)
	}
	return dump, nil
}
