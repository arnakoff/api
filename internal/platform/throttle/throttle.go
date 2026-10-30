// SPDX-License-Identifier: AGPL-3.0-only

// Package throttle keeps the registry clients away from an upstream that
// asked to wait with a Retry-After header (RFC 9110, section 10.2.3), so a
// check fails fast instead of calling it again too soon.
package throttle

import (
	"errors"
	"sync"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
)

// MaxWait caps a Retry-After, a choice: an absurd one must not keep an
// upstream unused until the next restart.
const MaxWait = time.Hour

// Throttle remembers, per host, until when an upstream asked not to be
// called. It is safe for concurrent use.
type Throttle struct {
	now func() time.Time

	mu    sync.Mutex
	until map[string]time.Time
}

// New returns a Throttle that dates the waits with now.
func New(now func() time.Time) *Throttle {
	return &Throttle{now: now, until: make(map[string]time.Time)}
}

// Allows reports whether host may be called now, and forgets a wait that
// is over.
func (t *Throttle) Allows(host string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	until, ok := t.until[host]
	if !ok {
		return true
	}
	if t.now().Before(until) {
		return false
	}
	delete(t.until, host)
	return true
}

// Wait keeps host from being called for d, at most MaxWait.
func (t *Throttle) Wait(host string, d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.until[host] = t.now().Add(min(d, MaxWait))
}

// Heed waits as long as err asks: an *httpclient.StatusError that carries
// a Retry-After. Any other error is ignored.
func (t *Throttle) Heed(host string, err error) {
	se, ok := errors.AsType[*httpclient.StatusError](err)
	if !ok {
		return
	}
	if d, ok := se.RetryAfter(); ok {
		t.Wait(host, d)
	}
}
