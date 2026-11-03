// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// keySecret is the HMAC key the API key tests digest with.
var keySecret = []byte("a test secret of thirty-two byte")

// keyStore is an in-memory app.KeyFinder and app.KeyIssuer. A non-nil err
// is the answer to every call.
type keyStore struct {
	mu   sync.Mutex
	recs []domain.KeyRecord
	err  error
}

func (s *keyStore) FindKey(_ context.Context, d domain.KeyDigest) (domain.KeyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return domain.KeyRecord{}, s.err
	}
	i := slices.IndexFunc(s.recs, func(r domain.KeyRecord) bool { return r.Digest == d })
	if i < 0 {
		return domain.KeyRecord{}, app.ErrKeyNotFound
	}
	return s.recs[i], nil
}

func (s *keyStore) CreateKey(_ context.Context, r domain.KeyRecord) (domain.KeyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return domain.KeyRecord{}, s.err
	}
	r.ID = int64(len(s.recs) + 1)
	s.recs = append(s.recs, r)
	return r, nil
}

func (s *keyStore) ListKeys(context.Context) ([]domain.KeyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	out := slices.Clone(s.recs)
	slices.SortFunc(out, func(a, b domain.KeyRecord) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (s *keyStore) RevokeKey(_ context.Context, id int64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	i := slices.IndexFunc(s.recs, func(r domain.KeyRecord) bool { return r.ID == id })
	if i < 0 {
		return app.ErrKeyNotFound
	}
	if s.recs[i].Active() {
		s.recs[i].RevokedAt = at
	}
	return nil
}

func (s *keyStore) PurgeRevokedKeys(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, s.err
	}
	before := len(s.recs)
	s.recs = slices.DeleteFunc(s.recs, func(r domain.KeyRecord) bool {
		return !r.Active() && r.RevokedAt.Before(cutoff)
	})
	return int64(before - len(s.recs)), nil
}
