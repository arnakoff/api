// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// hookedFinder is an app.KeyFinder that calls during at the start of each
// lookup, then passes it on to its keyStore.
type hookedFinder struct {
	store  *keyStore
	during func()
}

func (f *hookedFinder) FindKey(ctx context.Context, d domain.KeyDigest) (domain.KeyRecord, error) {
	f.during()
	return f.store.FindKey(ctx, d)
}

func TestKeyCacheCachesNothingUntilResumed(t *testing.T) {
	t.Parallel()

	d := domain.KeyDigest{1}
	store := &keyStore{recs: []domain.KeyRecord{{ID: 1, Digest: d, Environment: domain.KeyLive, CreatedAt: t0}}}
	finder := &countingFinder{store: store}
	cache := app.NewKeyCache(finder, &clock{now: t0}, 30*time.Second)
	findN(t, cache, d, 2, nil)
	if got := finder.finds.Load(); got != 2 {
		t.Errorf("store lookups before Resume = %d, want 2", got)
	}
}

func TestKeyCacheForgetsAKeyAtOnce(t *testing.T) {
	t.Parallel()

	cache, _, finder, _, d := newCachedKeys(t, 30*time.Second)
	findN(t, cache, d, 2, nil)
	cache.Forget(d)
	findN(t, cache, d, 2, nil)
	if got := finder.finds.Load(); got != 2 {
		t.Errorf("store lookups = %d, want 2: one before Forget, one after", got)
	}
}

func TestKeyCacheSuspendedForgetsEverythingAndCachesNothing(t *testing.T) {
	t.Parallel()

	cache, _, finder, _, d := newCachedKeys(t, 30*time.Second)
	findN(t, cache, d, 1, nil)
	cache.Suspend()
	findN(t, cache, d, 2, nil)
	if got := finder.finds.Load(); got != 3 {
		t.Errorf("store lookups while suspended = %d, want 3", got)
	}
	cache.Resume()
	findN(t, cache, d, 2, nil)
	if got := finder.finds.Load(); got != 4 {
		t.Errorf("store lookups after Resume = %d, want 4", got)
	}
}

// A record read before an invalidation that lands during the lookup may
// be stale: it is returned but not kept.
func TestKeyCacheKeepsNoRecordReadAcrossAnInvalidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		suspended  bool
		invalidate func(*app.KeyCache, domain.KeyDigest)
	}{
		{"forget", false, func(c *app.KeyCache, d domain.KeyDigest) { c.Forget(d) }},
		{"suspend then resume", false, func(c *app.KeyCache, _ domain.KeyDigest) { c.Suspend(); c.Resume() }},
		// A revocation committed before the watch listened is never told.
		{"resume", true, func(c *app.KeyCache, _ domain.KeyDigest) { c.Resume() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := domain.KeyDigest{1}
			store := &keyStore{recs: []domain.KeyRecord{{ID: 1, Digest: d, Environment: domain.KeyLive, CreatedAt: t0}}}
			finds := 0
			finder := &hookedFinder{store: store}
			cache := app.NewKeyCache(finder, &clock{now: t0}, 30*time.Second)
			if !tt.suspended {
				cache.Resume()
			}
			finder.during = func() {
				finds++
				if finds == 1 {
					tt.invalidate(cache, d)
				}
			}
			findN(t, cache, d, 3, nil)
			if finds != 3-1 {
				t.Errorf("store lookups = %d, want 2: the first record is not kept", finds)
			}
		})
	}
}
