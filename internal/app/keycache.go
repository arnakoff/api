// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"sync"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// KeyCacheTTL is how long a KeyCache keeps an active key: a key revoked in
// the store is still accepted for at most this long. Validated by the
// owner on 02/11/2026 (#198).
const KeyCacheTTL = 30 * time.Second

// KeyCache is a KeyFinder keeping in memory, for a fixed time, the active
// keys its store finds, so that a valid key does not cost a store lookup
// on each request. It keeps nothing else: an unknown or revoked key and a
// store failure reach the store each time, so a revocation is seen as soon
// as the cached record expires and an outage is never hidden. Its size is
// bounded by the number of active keys issued, not by what clients send.
type KeyCache struct {
	keys  KeyFinder
	clock Clock
	ttl   time.Duration

	mu      sync.RWMutex
	entries map[domain.KeyDigest]cachedKey
}

// cachedKey is an active record and when it stops being served.
type cachedKey struct {
	rec     domain.KeyRecord
	expires time.Time
}

// NewKeyCache returns a KeyCache over keys keeping each active record for
// ttl, as told by clock.
func NewKeyCache(keys KeyFinder, clock Clock, ttl time.Duration) *KeyCache {
	return &KeyCache{keys: keys, clock: clock, ttl: ttl, entries: make(map[domain.KeyDigest]cachedKey)}
}

// FindKey returns the record of the key of digest d, from memory while it
// is fresh, or else from the store.
func (c *KeyCache) FindKey(ctx context.Context, d domain.KeyDigest) (domain.KeyRecord, error) {
	now := c.clock.Now()
	if rec, ok := c.fresh(d, now); ok {
		return rec, nil
	}
	rec, err := c.keys.FindKey(ctx, d)
	c.keep(d, rec, err, now)
	return rec, err
}

// fresh returns the cached record of d if it has not expired at now.
func (c *KeyCache) fresh(d domain.KeyDigest, now time.Time) (domain.KeyRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[d]
	if !ok || !now.Before(e.expires) {
		return domain.KeyRecord{}, false
	}
	return e.rec, true
}

// keep caches the answer of the store for d, fetched at now, when it is an
// active record, and drops any previous entry of d otherwise.
func (c *KeyCache) keep(d domain.KeyDigest, rec domain.KeyRecord, err error, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil || !rec.Active() {
		delete(c.entries, d)
		return
	}
	c.entries[d] = cachedKey{rec: rec, expires: now.Add(c.ttl)}
}
