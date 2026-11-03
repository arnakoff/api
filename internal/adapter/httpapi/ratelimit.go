// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/arnakoff/api/internal/platform/ratelimit"
)

// Limiter limits the requests of each client, known by a key.
type Limiter interface {
	// Allow spends one request of key.
	Allow(key string) ratelimit.Decision
	// Policy returns the policy applied to each key.
	Policy() ratelimit.Policy
	// Refund gives back one request of key, spent by Allow.
	Refund(key string)
}

// Names of the quota policies in the RateLimit fields.
const (
	keyPolicy = "key"
	ipPolicy  = "ip"
	// authPolicy limits the refused API keys of each client address.
	authPolicy = "auth"
)

// Canonical forms of the RateLimit field names
// (textproto.CanonicalMIMEHeaderKey): admit sets them directly, which
// spares canonicalising them on every answer.
const (
	rateLimitPolicyField = "Ratelimit-Policy"
	rateLimitField       = "Ratelimit"
)

// quota applies a Limiter under a policy name.
type quota struct {
	limiter Limiter
	name    string
	// policy is the RateLimit-Policy field, the same on every answer.
	policy string
	ps     problems
}

// newQuota returns the quota of l under name.
func newQuota(l Limiter, name string, ps problems) *quota {
	p := l.Policy()
	policy := `"` + name + `";q=` + strconv.Itoa(p.Quota) + ";w=" + seconds(p.Window)
	return &quota{limiter: l, name: name, policy: policy, ps: ps}
}

// admit spends one request of key and reports whether it may be served;
// otherwise it answers a 429 with Retry-After. Either way the answer
// carries the RateLimit-Policy and RateLimit fields
// (draft-ietf-httpapi-ratelimit-headers-11, sections 3 and 4).
func (q *quota) admit(w http.ResponseWriter, r *http.Request, key string) bool {
	d := q.limiter.Allow(key)
	if !d.Allowed {
		q.deny(w, r, d)
		return false
	}
	q.describe(w.Header(), d)
	return true
}

// deny answers the 429 of d, a request q refused, with Retry-After and the
// RateLimit fields.
func (q *quota) deny(w http.ResponseWriter, r *http.Request, d ratelimit.Decision) {
	h := w.Header()
	q.describe(h, d)
	h.Set("Retry-After", seconds(d.Reset))
	q.ps.write(w, r, tooManyRequests)
}

// describe sets the RateLimit-Policy and RateLimit fields of d in h.
func (q *quota) describe(h http.Header, d ratelimit.Decision) {
	h[rateLimitPolicyField] = []string{q.policy}
	h[rateLimitField] = []string{`"` + q.name + `";r=` + strconv.Itoa(d.Remaining) + ";t=" + seconds(d.Reset)}
}

// limit answers next only for a request q admits, the client known by
// keyOf. A limiter too full to track one more client lets that client be
// served, with no request left in its RateLimit field: a flood from many
// addresses must not lock every new client out of routes that only read
// memory.
func limit(q *quota, keyOf func(*http.Request) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := q.limiter.Allow(keyOf(r))
		if !d.Allowed && !d.Full {
			q.deny(w, r, d)
			return
		}
		q.describe(w.Header(), d)
		next.ServeHTTP(w, r)
	})
}

// limiterOr returns l, or when it is nil a limiter of n requests a minute
// per client: the API always limits.
func limiterOr(l Limiter, n int) Limiter {
	if l != nil {
		return l
	}
	return ratelimit.PerMinute(n, time.Now)
}

// seconds writes d in whole seconds, as delay-seconds (RFC 9110, section
// 10.2.3).
func seconds(d time.Duration) string {
	return strconv.FormatInt(int64(d/time.Second), 10)
}
