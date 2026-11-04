// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// countingFinder is an app.KeyFinder counting the lookups it passes on to
// its keyStore.
type countingFinder struct {
	store *keyStore
	finds atomic.Int64
}

func (f *countingFinder) FindKey(ctx context.Context, d domain.KeyDigest) (domain.KeyRecord, error) {
	f.finds.Add(1)
	return f.store.FindKey(ctx, d)
}

// newCachedKeys returns a KeyCache of ttl over a store holding one active
// key issued at t0, resumed, the store, its counter, the clock and the key
// digest.
func newCachedKeys(t *testing.T, ttl time.Duration) (*app.KeyCache, *keyStore, *countingFinder, *clock, domain.KeyDigest) {
	t.Helper()
	d := domain.KeyDigest{1}
	store := &keyStore{recs: []domain.KeyRecord{{ID: 1, Digest: d, Environment: domain.KeyLive, CreatedAt: t0}}}
	finder := &countingFinder{store: store}
	c := &clock{now: t0}
	cache := app.NewKeyCache(finder, c, ttl)
	cache.Resume()
	return cache, store, finder, c, d
}

// findN calls FindKey n times and fails on an error other than want.
func findN(t *testing.T, cache *app.KeyCache, d domain.KeyDigest, n int, want error) {
	t.Helper()
	for range n {
		if _, err := cache.FindKey(context.Background(), d); !errors.Is(err, want) {
			t.Fatalf("FindKey() error = %v, want %v", err, want)
		}
	}
}

func TestKeyCacheServesAnActiveKeyFromMemoryWithinTheTTL(t *testing.T) {
	t.Parallel()

	cache, _, finder, c, d := newCachedKeys(t, 30*time.Second)
	findN(t, cache, d, 3, nil)
	c.advance(30*time.Second - time.Nanosecond)
	rec, err := cache.FindKey(context.Background(), d)
	if err != nil || rec.ID != 1 {
		t.Fatalf("FindKey() = %+v, %v, want the record of key 1", rec, err)
	}
	if got := finder.finds.Load(); got != 1 {
		t.Errorf("store lookups = %d, want 1", got)
	}
}

func TestKeyCacheSeesARevocationOnceTheTTLIsOver(t *testing.T) {
	t.Parallel()

	cache, store, finder, c, d := newCachedKeys(t, 30*time.Second)
	findN(t, cache, d, 1, nil)
	if err := store.RevokeKey(context.Background(), 1, t0); err != nil {
		t.Fatal(err)
	}
	c.advance(30 * time.Second)
	rec, err := cache.FindKey(context.Background(), d)
	if err != nil || rec.Active() {
		t.Fatalf("FindKey() after the TTL = %+v, %v, want the revoked record", rec, err)
	}
	findN(t, cache, d, 2, nil)
	if got := finder.finds.Load(); got != 4 {
		t.Errorf("store lookups = %d, want 4: a revoked key is never cached", got)
	}
}

func TestKeyCacheKeepsNoRefusalOrFailure(t *testing.T) {
	t.Parallel()

	cache, store, finder, c, d := newCachedKeys(t, 30*time.Second)
	findN(t, cache, domain.KeyDigest{2}, 2, app.ErrKeyNotFound)
	findN(t, cache, d, 1, nil)

	c.advance(30 * time.Second)
	store.err = errors.New("connection refused")
	findN(t, cache, d, 2, store.err)

	store.err = nil
	store.recs = nil
	findN(t, cache, d, 2, app.ErrKeyNotFound)
	if got := finder.finds.Load(); got != 7 {
		t.Errorf("store lookups = %d, want 7: only an active record is cached", got)
	}
}

func TestKeyCacheIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	cache, _, _, c, d := newCachedKeys(t, 30*time.Second)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				if _, err := cache.FindKey(context.Background(), d); err != nil {
					t.Errorf("FindKey() error = %v", err)
					return
				}
				c.advance(time.Second)
			}
		})
	}
	wg.Wait()
}
