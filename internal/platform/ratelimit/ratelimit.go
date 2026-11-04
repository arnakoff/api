// SPDX-License-Identifier: AGPL-3.0-only

// Package ratelimit limits the requests of each client with a token bucket
// per key, in memory and bounded: an idle bucket is forgotten, and the
// number of buckets is capped. A spent request can be given back, which
// golang.org/x/time/rate cannot do once time has passed
// (Reservation.CancelAt restores nothing after the time to act).
package ratelimit

import (
	"math"
	"sync"
	"time"
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
	// Full reports a key refused only because the Limiter already keeps
	// as many keys as it may: the key itself spent nothing.
	Full bool
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

// bucket is the token bucket of a key: the requests it held when it was
// last used, at seen.
type bucket struct {
	tokens float64
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

// PerHour returns a limiter allowing each client n requests an hour,
// keeping at most DefaultMaxKeys clients.
func PerHour(n int, now func() time.Time) *Limiter {
	return New(Policy{Quota: n, Window: time.Hour}, DefaultMaxKeys, now)
}

// Policy returns the policy l applies.
func (l *Limiter) Policy() Policy {
	return l.policy
}

// Allow spends one request of key. A key l does not know yet while it
// already keeps maxKeys buckets is refused for a whole window: memory stays
// bounded, and the keys already known keep their quota. Allow looks its
// bucket up itself rather than through bucketOf, which does not inline: a
// call on every request costs it 10 % (BenchmarkAllow).
func (l *Limiter) Allow(key string) Decision {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) >= l.policy.Window {
		l.sweep(now)
	}
	b, ok := l.buckets[key]
	if !ok {
		if b, ok = l.add(key, now); !ok {
			return Decision{Reset: l.policy.Window, Full: true}
		}
	}
	l.refill(b, now)
	allowed := b.tokens >= 1
	if allowed {
		b.tokens--
	}
	return l.decision(b.tokens, allowed)
}

// Peek returns whether key has a request left, spending none. A key l
// does not know has its whole quota: Peek keeps nothing for it.
func (l *Limiter) Peek(key string) Decision {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		return Decision{Allowed: true, Remaining: l.policy.Quota}
	}
	l.refill(b, now)
	return l.decision(b.tokens, b.tokens >= 1)
}

// Charge spends one request of key, even one it does not have: its bucket
// then owes it, and Allow and Peek refuse key until the debt is repaid at
// the pace of the policy. Requests served before they are charged, many
// at once, are so all counted. A key l does not know yet while it already
// keeps maxKeys buckets is charged nothing.
func (l *Limiter) Charge(key string) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.bucketOf(key, now)
	if !ok {
		return
	}
	l.refill(b, now)
	b.tokens--
}

// bucketOf returns the bucket of key, made by add when l does not know
// key, and false when l keeps too many buckets to make one. It sweeps l
// once a window.
func (l *Limiter) bucketOf(key string, now time.Time) (*bucket, bool) {
	if now.Sub(l.lastSweep) >= l.policy.Window {
		l.sweep(now)
	}
	if b, ok := l.buckets[key]; ok {
		return b, true
	}
	return l.add(key, now)
}

// add makes the bucket of key with the whole quota, and returns false when
// l keeps too many buckets to make one, once it swept for room at most
// once a minSweepGap.
func (l *Limiter) add(key string, now time.Time) (*bucket, bool) {
	if len(l.buckets) >= l.maxKeys && now.Sub(l.lastSweep) >= minSweepGap {
		l.sweep(now)
	}
	if len(l.buckets) >= l.maxKeys {
		return nil, false
	}
	b := &bucket{tokens: float64(l.policy.Quota), seen: now}
	l.buckets[key] = b
	return b, true
}

// decision is the Decision of a bucket holding tokens, owing requests
// when they are negative.
func (l *Limiter) decision(tokens float64, allowed bool) Decision {
	return Decision{Allowed: allowed, Remaining: max(int(tokens), 0), Reset: l.untilNext(tokens)}
}

// Refund gives back one request of key, spent by Allow, up to the quota. A
// key l does not know has its whole quota: Refund keeps nothing for it. A
// refunded bucket is kept until the sweep, so that a client whose requests
// are all refunded does not allocate one on each.
func (l *Limiter) Refund(key string) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		return
	}
	l.refill(b, now)
	b.tokens = min(b.tokens+1, float64(l.policy.Quota))
}

// refill restores the requests b earned since it was last used, up to the
// quota, and dates it now. A clock going back restores nothing.
func (l *Limiter) refill(b *bucket, now time.Time) {
	if elapsed := now.Sub(b.seen); elapsed > 0 {
		b.tokens = min(b.tokens+elapsed.Seconds()*l.rate(), float64(l.policy.Quota))
	}
	b.seen = now
}

// sweep forgets the buckets idle for a window and full again, as is any
// such bucket owing no request: forgetting them changes no decision. A
// bucket in debt is kept until repaid.
func (l *Limiter) sweep(now time.Time) {
	for key, b := range l.buckets {
		idle := now.Sub(b.seen)
		if idle >= l.policy.Window && b.tokens+idle.Seconds()*l.rate() >= float64(l.policy.Quota) {
			delete(l.buckets, key)
		}
	}
	l.lastSweep = now
}

// rate is the pace at which the policy restores requests, per second.
func (l *Limiter) rate() float64 {
	return float64(l.policy.Quota) / l.policy.Window.Seconds()
}

// untilNext returns how long a bucket holding tokens takes to restore one
// more whole request, or when in debt to repay it and restore one, rounded
// up to the second; zero when it is full.
func (l *Limiter) untilNext(tokens float64) time.Duration {
	if tokens >= float64(l.policy.Quota) {
		return 0
	}
	missing := max(math.Floor(tokens)+1, 1) - tokens
	return time.Duration(math.Ceil(missing/l.rate())) * time.Second
}
