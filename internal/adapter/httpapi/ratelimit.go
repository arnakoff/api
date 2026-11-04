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
	// AllowN spends n requests of key, or none when it has fewer.
	AllowN(key string, n int) ratelimit.Decision
	// Policy returns the policy applied to each key.
	Policy() ratelimit.Policy
	// Refund gives back one request of key, spent by Allow.
	Refund(key string)
}

// RefusalLimiter limits the refusals of each client, known by a key,
// counted once they happen.
type RefusalLimiter interface {
	// Peek returns whether key may be refused once more, spending nothing.
	Peek(key string) ratelimit.Decision
	// Charge spends one refusal of key, even one it does not have.
	Charge(key string)
	// Policy returns the policy applied to each key.
	Policy() ratelimit.Policy
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
	fields
}

// newQuota returns the quota of l under name.
func newQuota(l Limiter, name string, ps problems) *quota {
	return &quota{limiter: l, fields: newFields(l.Policy(), name, ps)}
}

// fields writes the RateLimit fields, and the 429, of a policy name.
type fields struct {
	name string
	// policy is the RateLimit-Policy field, the same on every answer.
	policy string
	ps     problems
}

// newFields returns the fields of p under name.
func newFields(p ratelimit.Policy, name string, ps problems) fields {
	policy := `"` + name + `";q=` + strconv.Itoa(p.Quota) + ";w=" + seconds(p.Window)
	return fields{name: name, policy: policy, ps: ps}
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

// admitMore makes the request of key, which admit let in for one request,
// spend n instead, all or none: the request is then served, with the
// RateLimit fields of what is left. Otherwise it answers a 429 and spends
// nothing, the request admit spent given back.
func (q *quota) admitMore(w http.ResponseWriter, r *http.Request, key string, n int) bool {
	if n <= 1 {
		return true
	}
	q.limiter.Refund(key)
	d := q.limiter.AllowN(key, n)
	if !d.Allowed {
		q.deny(w, r, d)
		return false
	}
	q.describe(w.Header(), d)
	return true
}

// deny answers the 429 of d, a request f refused, with Retry-After and the
// RateLimit fields.
func (f fields) deny(w http.ResponseWriter, r *http.Request, d ratelimit.Decision) {
	h := w.Header()
	f.describe(h, d)
	h.Set("Retry-After", seconds(d.Reset))
	f.ps.write(w, r, tooManyRequests)
}

// describe sets the RateLimit-Policy and RateLimit fields of d in h.
func (f fields) describe(h http.Header, d ratelimit.Decision) {
	h[rateLimitPolicyField] = []string{f.policy}
	h[rateLimitField] = []string{`"` + f.name + `";r=` + strconv.Itoa(d.Remaining) + ";t=" + seconds(d.Reset)}
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

// refusalLimiterOr returns l, or when it is nil a limiter of n refusals a
// minute per client.
func refusalLimiterOr(l RefusalLimiter, n int) RefusalLimiter {
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
