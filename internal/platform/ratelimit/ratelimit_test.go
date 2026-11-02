// SPDX-License-Identifier: AGPL-3.0-only

package ratelimit_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/ratelimit"
)

// fakeClock is a settable clock.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)}
}

// perMinute is the policy of the tests: 60 requests a minute, one second
// to restore each.
var perMinute = ratelimit.Policy{Quota: 60, Window: time.Minute}

func TestAllowCountsDownThenRefuses(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(perMinute, 10, newClock().Now)
	for i := range perMinute.Quota {
		d := l.Allow("k")
		want := ratelimit.Decision{Allowed: true, Remaining: perMinute.Quota - 1 - i, Reset: time.Second}
		if d != want {
			t.Fatalf("Allow() #%d = %+v, want %+v", i+1, d, want)
		}
	}
	want := ratelimit.Decision{Allowed: false, Remaining: 0, Reset: time.Second}
	if d := l.Allow("k"); d != want {
		t.Errorf("Allow() over the quota = %+v, want %+v", d, want)
	}
}

func TestAllowRestoresOverTime(t *testing.T) {
	t.Parallel()
	clk := newClock()
	l := ratelimit.New(perMinute, 10, clk.Now)
	for range perMinute.Quota {
		l.Allow("k")
	}
	clk.advance(400 * time.Millisecond)
	if d := l.Allow("k"); d.Allowed || d.Reset != time.Second {
		t.Errorf("Allow() 0.4 s later = %+v, want refused, Reset rounded up to 1 s", d)
	}
	clk.advance(600 * time.Millisecond)
	if d := l.Allow("k"); !d.Allowed || d.Remaining != 0 {
		t.Errorf("Allow() 1 s later = %+v, want allowed with nothing left", d)
	}
	clk.advance(perMinute.Window)
	want := ratelimit.Decision{Allowed: true, Remaining: perMinute.Quota - 1, Reset: time.Second}
	if d := l.Allow("k"); d != want {
		t.Errorf("Allow() a window later = %+v, want %+v", d, want)
	}
}

func TestAllowKeepsKeysApart(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(ratelimit.Policy{Quota: 1, Window: time.Minute}, 10, newClock().Now)
	if !l.Allow("a").Allowed || l.Allow("a").Allowed {
		t.Fatal("key a: want one request allowed, then refused")
	}
	if !l.Allow("b").Allowed {
		t.Error("Allow(b) after a ran out = refused, want allowed")
	}
}

func TestAllowForgetsIdleKeys(t *testing.T) {
	t.Parallel()
	clk := newClock()
	l := ratelimit.New(perMinute, 10, clk.Now)
	l.Allow("a")
	l.Allow("b")
	clk.advance(perMinute.Window)
	l.Allow("c")
	if n := ratelimit.Len(l); n != 1 {
		t.Errorf("keys kept a window later = %d, want 1", n)
	}
}

func TestAllowRefusesNewKeysOverTheCap(t *testing.T) {
	t.Parallel()
	clk := newClock()
	l := ratelimit.New(perMinute, 2, clk.Now)
	l.Allow("a")
	l.Allow("b")
	want := ratelimit.Decision{Allowed: false, Remaining: 0, Reset: perMinute.Window}
	if d := l.Allow("c"); d != want {
		t.Errorf("Allow(c) over the cap = %+v, want %+v", d, want)
	}
	if !l.Allow("a").Allowed {
		t.Error("Allow(a), a known key, over the cap = refused, want allowed")
	}
	clk.advance(perMinute.Window)
	if !l.Allow("c").Allowed {
		t.Error("Allow(c) once the idle keys are gone = refused, want allowed")
	}
}

func TestAFullLimiterSweepsToMakeRoom(t *testing.T) {
	t.Parallel()
	clk := newClock()
	l := ratelimit.New(perMinute, 2, clk.Now)
	clk.advance(30 * time.Second)
	l.Allow("a")
	l.Allow("b")
	clk.advance(30 * time.Second)
	// The periodic sweep runs, but a and b are idle for only 30 s.
	if l.Allow("x").Allowed {
		t.Fatal("Allow(x) over the cap = allowed, want refused")
	}
	clk.advance(30 * time.Second)
	// No periodic sweep is due; the full limiter sweeps for room.
	if !l.Allow("c").Allowed {
		t.Error("Allow(c) once a and b are idle a window = refused, want allowed")
	}
}

func TestAllowIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(perMinute, 1000, newClock().Now)
	var wg sync.WaitGroup
	allowed := make(chan bool, 4*perMinute.Quota)
	for i := range 4 * perMinute.Quota {
		wg.Go(func() { allowed <- l.Allow(strconv.Itoa(i % 2)).Allowed })
	}
	wg.Wait()
	close(allowed)
	n := 0
	for ok := range allowed {
		if ok {
			n++
		}
	}
	if n != 2*perMinute.Quota {
		t.Errorf("allowed = %d, want %d: the quota of each of 2 keys", n, 2*perMinute.Quota)
	}
}

func TestPolicy(t *testing.T) {
	t.Parallel()
	if p := ratelimit.New(perMinute, 1, newClock().Now).Policy(); p != perMinute {
		t.Errorf("Policy() = %+v, want %+v", p, perMinute)
	}
}

func TestPerMinuteAppliesTheQuota(t *testing.T) {
	t.Parallel()

	if p := ratelimit.PerMinute(42, time.Now).Policy(); p.Quota != 42 || p.Window != time.Minute {
		t.Errorf("PerMinute(42).Policy() = %+v, want 42 a minute", p)
	}
}
