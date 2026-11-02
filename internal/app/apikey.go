// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// API key errors, checked with errors.Is. Any other error of a key store
// means it could not answer.
var (
	// ErrKeyNotFound means no key has this digest, or this identifier.
	ErrKeyNotFound = errors.New("API key not found")
	// ErrKeyRevoked means the key was revoked.
	ErrKeyRevoked = errors.New("API key revoked")
)

// KeyFinder finds an issued API key by its digest.
type KeyFinder interface {
	// FindKey returns the record of the key of digest d, or ErrKeyNotFound.
	FindKey(ctx context.Context, d domain.KeyDigest) (domain.KeyRecord, error)
}

// KeyIssuer keeps the records of issued API keys.
type KeyIssuer interface {
	// CreateKey stores r and returns it with its identifier set.
	CreateKey(ctx context.Context, r domain.KeyRecord) (domain.KeyRecord, error)
	// ListKeys returns every record, oldest first.
	ListKeys(ctx context.Context) ([]domain.KeyRecord, error)
	// RevokeKey marks the key id revoked at at, or returns ErrKeyNotFound.
	// A key already revoked keeps its first revocation time.
	RevokeKey(ctx context.Context, id int64, at time.Time) error
}

// Authenticator tells whether an API key may call the API.
type Authenticator struct {
	keys   KeyFinder
	secret []byte
	env    domain.KeyEnvironment
}

// NewAuthenticator returns an Authenticator accepting the active keys of
// env found in keys, digested with secret.
func NewAuthenticator(keys KeyFinder, secret []byte, env domain.KeyEnvironment) *Authenticator {
	return &Authenticator{keys: keys, secret: slices.Clone(secret), env: env}
}

// Authenticate returns the record of the key raw. It fails with
// domain.ErrInvalidAPIKey for a malformed key, ErrKeyNotFound for a key
// never issued or issued for another environment, ErrKeyRevoked for a
// revoked key, and any other error when the store could not answer.
func (a *Authenticator) Authenticate(ctx context.Context, raw string) (domain.KeyRecord, error) {
	key, err := domain.ParseAPIKey(raw)
	if err != nil {
		return domain.KeyRecord{}, err
	}
	if key.Environment() != a.env {
		return domain.KeyRecord{}, ErrKeyNotFound
	}
	d := domain.DigestAPIKey(a.secret, key)
	rec, err := a.keys.FindKey(ctx, d)
	switch {
	case errors.Is(err, ErrKeyNotFound):
		return domain.KeyRecord{}, err
	case err != nil:
		return domain.KeyRecord{}, fmt.Errorf("find API key %s: %w", key.Redacted(), err)
	case subtle.ConstantTimeCompare(rec.Digest[:], d[:]) != 1:
		// The store matched on an index: compare again, in constant time.
		return domain.KeyRecord{}, ErrKeyNotFound
	case !rec.Active():
		return domain.KeyRecord{}, ErrKeyRevoked
	}
	return rec, nil
}

// KeyAdmin issues, lists and revokes API keys.
type KeyAdmin struct {
	keys   KeyIssuer
	secret []byte
	clock  Clock
	random io.Reader
}

// NewKeyAdmin returns a KeyAdmin storing in keys the digests under secret
// of keys drawn from random, such as crypto/rand.Reader.
func NewKeyAdmin(keys KeyIssuer, secret []byte, clock Clock, random io.Reader) *KeyAdmin {
	return &KeyAdmin{keys: keys, secret: slices.Clone(secret), clock: clock, random: random}
}

// Issue draws a key of env for the holder label, stores its record and
// returns both: the key is shown once and never kept.
func (a *KeyAdmin) Issue(ctx context.Context, env domain.KeyEnvironment, label string) (domain.APIKey, domain.KeyRecord, error) {
	label, err := domain.ParseKeyLabel(label)
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, err
	}
	key, err := domain.NewAPIKey(env, a.random)
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, err
	}
	rec, err := a.keys.CreateKey(ctx, domain.KeyRecord{
		Digest: domain.DigestAPIKey(a.secret, key), Environment: env,
		Label: label, Hint: key.Hint(), CreatedAt: a.clock.Now().UTC(),
	})
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, fmt.Errorf("store API key: %w", err)
	}
	return key, rec, nil
}

// List returns the record of every issued key, oldest first.
func (a *KeyAdmin) List(ctx context.Context) ([]domain.KeyRecord, error) {
	recs, err := a.keys.ListKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list API keys: %w", err)
	}
	return recs, nil
}

// Revoke revokes the key id now, or returns ErrKeyNotFound.
func (a *KeyAdmin) Revoke(ctx context.Context, id int64) error {
	if err := a.keys.RevokeKey(ctx, id, a.clock.Now().UTC()); err != nil {
		return fmt.Errorf("revoke API key %d: %w", id, err)
	}
	return nil
}
