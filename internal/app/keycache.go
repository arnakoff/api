// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"sync"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// KeyCacheTTL is how long a KeyCache keeps an active key. Revocations are
// forgotten at once (Forget); the TTL only bounds how long a record lives
// in memory. Validated by the owner on 02/11/2026 (#198).
const KeyCacheTTL = 30 * time.Second

// KeyCache is a KeyFinder keeping in memory, for a fixed time, the active
// keys its store finds, so that a valid key does not cost a store lookup
// on each request. It keeps nothing else: an unknown or revoked key and a
// store failure reach the store each time, and an outage is never hidden.
// Its size is bounded by the number of active keys issued, not by what
// clients send.
//
// A revoked key must be forgotten at once with Forget, on every instance.
// The cache therefore starts suspended and caches nothing until Resume,
// which the watcher of revocations calls once it is sure to hear them;
// it calls Suspend as soon as it may miss one.
type KeyCache struct {
	keys  KeyFinder
	clock Clock
	ttl   time.Duration

	mu      sync.RWMutex
	entries map[domain.KeyDigest]cachedKey
	active  bool
	// gen changes at each invalidation: a record read under an older
	// generation may predate a revocation and is not kept.
	gen uint64
}

// cachedKey is an active record and when it stops being served.
type cachedKey struct {
	rec     domain.KeyRecord
	expires time.Time
}

// NewKeyCache returns a suspended KeyCache over keys keeping each active
// record for ttl, as told by clock, once resumed.
func NewKeyCache(keys KeyFinder, clock Clock, ttl time.Duration) *KeyCache {
	return &KeyCache{keys: keys, clock: clock, ttl: ttl, entries: make(map[domain.KeyDigest]cachedKey)}
}

// FindKey returns the record of the key of digest d, from memory while it
// is fresh, or else from the store.
func (c *KeyCache) FindKey(ctx context.Context, d domain.KeyDigest) (domain.KeyRecord, error) {
	now := c.clock.Now()
	rec, gen, ok := c.fresh(d, now)
	if ok {
		return rec, nil
	}
	rec, err := c.keys.FindKey(ctx, d)
	c.keep(d, rec, err, now, gen)
	return rec, err
}

// Forget drops the key of digest d, so that its next lookup reaches the
// store, as does any lookup already under way.
func (c *KeyCache) Forget(d domain.KeyDigest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, d)
	c.gen++
}

// Suspend drops every key and caches none until Resume.
func (c *KeyCache) Suspend() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.entries)
	c.active = false
	c.gen++
}

// Resume caches the active keys again, but none read before: a revocation
// may have been missed until then.
func (c *KeyCache) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active = true
	c.gen++
}

// fresh returns the cached record of d if it has not expired at now, and
// the generation a lookup of the store starts under.
func (c *KeyCache) fresh(d domain.KeyDigest, now time.Time) (domain.KeyRecord, uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[d]
	if !ok || !now.Before(e.expires) {
		return domain.KeyRecord{}, c.gen, false
	}
	return e.rec, c.gen, true
}

// keep caches the answer of the store for d, fetched at now under
// generation gen, when it is an active record read since the last
// invalidation and the cache is active, and drops any previous entry of d
// otherwise.
func (c *KeyCache) keep(d domain.KeyDigest, rec domain.KeyRecord, err error, now time.Time, gen uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil || !rec.Active() || !c.active || gen != c.gen {
		delete(c.entries, d)
		return
	}
	c.entries[d] = cachedKey{rec: rec, expires: now.Add(c.ttl)}
}
