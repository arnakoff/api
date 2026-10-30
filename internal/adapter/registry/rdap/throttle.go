// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"sync"
	"time"
)

// maxThrottle caps a Retry-After, a choice: an absurd one must not keep a
// whole extension without RDAP until the next restart.
const maxThrottle = time.Hour

// throttle remembers, per host, until when a registry asked not to be
// called. It is safe for concurrent use.
type throttle struct {
	now func() time.Time

	mu    sync.Mutex
	until map[string]time.Time
}

func newThrottle(now func() time.Time) *throttle {
	return &throttle{now: now, until: make(map[string]time.Time)}
}

// allows reports whether host may be called now, and forgets a wait that
// is over.
func (t *throttle) allows(host string) bool {
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

// wait keeps host from being called for d, at most maxThrottle.
func (t *throttle) wait(host string, d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.until[host] = t.now().Add(min(d, maxThrottle))
}
