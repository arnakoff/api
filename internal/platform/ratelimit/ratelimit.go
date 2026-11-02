// SPDX-License-Identifier: AGPL-3.0-only

// Package ratelimit limits the requests of each client with a token bucket
// per key (golang.org/x/time/rate), in memory and bounded: an idle bucket
// is forgotten, and the number of buckets is capped.
package ratelimit

import (
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DefaultMaxKeys is a cap on the keys a Limiter tracks at once: 17 MB of
// heap, as measured with IPv6 prefixes as keys.
const DefaultMaxKeys = 100_000

// minSweepGap spaces the sweeps a full Limiter runs to make room for a new
// key: a flood of new keys must not scan every bucket on each request,
// under the lock.
const minSweepGap = time.Second

// Policy is a quota of requests per window. A client may spend the whole
// quota at once; it is then restored at an even pace over the window.
type Policy struct {
	// Quota is the number of requests of a window, at least one.
	Quota int
	// Window is the period over which Quota is restored, at least one
	// second.
	Window time.Duration
}

// Decision is the answer of a Limiter to one request.
type Decision struct {
	// Allowed reports whether the request may be served.
	Allowed bool
	// Remaining is the number of requests the client may still send now.
	Remaining int
	// Reset is how long until one more request is restored, rounded up to
	// the second: within it, the client may send no more than Remaining.
	// It is zero when the whole quota is available.
	Reset time.Duration
}

// Limiter applies a Policy to each key on its own. It is safe for
// concurrent use.
type Limiter struct {
	policy  Policy
	maxKeys int
	now     func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

// bucket is the token bucket of a key and when it was last used.
type bucket struct {
	tokens *rate.Limiter
	seen   time.Time
}

// New returns a Limiter applying p to at most maxKeys keys at once, dated
// with now.
func New(p Policy, maxKeys int, now func() time.Time) *Limiter {
	return &Limiter{
		policy: p, maxKeys: maxKeys, now: now,
		buckets: make(map[string]*bucket), lastSweep: now(),
	}
}

// PerMinute returns a limiter allowing each client n requests a minute,
// keeping at most DefaultMaxKeys clients.
func PerMinute(n int, now func() time.Time) *Limiter {
	return New(Policy{Quota: n, Window: time.Minute}, DefaultMaxKeys, now)
}

// Policy returns the policy l applies.
func (l *Limiter) Policy() Policy {
	return l.policy
}

// Allow spends one request of key. A key l does not know yet while it
// already keeps maxKeys buckets is refused for a whole window: memory stays
// bounded, and the keys already known keep their quota.
func (l *Limiter) Allow(key string) Decision {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) >= l.policy.Window {
		l.sweep(now)
	}
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maxKeys && now.Sub(l.lastSweep) >= minSweepGap {
			l.sweep(now)
		}
		if len(l.buckets) >= l.maxKeys {
			return Decision{Reset: l.policy.Window}
		}
		b = &bucket{tokens: rate.NewLimiter(l.rate(), l.policy.Quota)}
		l.buckets[key] = b
	}
	b.seen = now
	allowed := b.tokens.AllowN(now, 1)
	left := b.tokens.TokensAt(now)
	return Decision{Allowed: allowed, Remaining: int(left), Reset: l.untilNext(left)}
}

// sweep forgets the buckets idle for a window. A bucket idle for a window
// is full again: forgetting it changes no decision.
func (l *Limiter) sweep(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.seen) >= l.policy.Window {
			delete(l.buckets, key)
		}
	}
	l.lastSweep = now
}

// rate is the pace at which the policy restores requests.
func (l *Limiter) rate() rate.Limit {
	return rate.Limit(float64(l.policy.Quota) / l.policy.Window.Seconds())
}

// untilNext returns how long a bucket holding tokens takes to restore one
// more whole request, rounded up to the second; zero when it is full.
func (l *Limiter) untilNext(tokens float64) time.Duration {
	if tokens >= float64(l.policy.Quota) {
		return 0
	}
	missing := math.Floor(tokens) + 1 - tokens
	return time.Duration(math.Ceil(missing/float64(l.rate()))) * time.Second
}
