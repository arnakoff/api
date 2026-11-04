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
	want := ratelimit.Decision{Allowed: false, Remaining: 0, Reset: perMinute.Window, Full: true}
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

func TestPerHourAppliesTheQuota(t *testing.T) {
	t.Parallel()

	if p := ratelimit.PerHour(3, time.Now).Policy(); p.Quota != 3 || p.Window != time.Hour {
		t.Errorf("PerHour(3).Policy() = %+v, want 3 an hour", p)
	}
}

func TestPerMinuteAppliesTheQuota(t *testing.T) {
	t.Parallel()

	if p := ratelimit.PerMinute(42, time.Now).Policy(); p.Quota != 42 || p.Window != time.Minute {
		t.Errorf("PerMinute(42).Policy() = %+v, want 42 a minute", p)
	}
}

func TestRefundGivesBackASpentRequest(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(ratelimit.Policy{Quota: 2, Window: time.Minute}, 10, newClock().Now)
	l.Allow("k")
	l.Allow("k")
	l.Refund("k")
	want := ratelimit.Decision{Allowed: true, Remaining: 0, Reset: 30 * time.Second}
	if d := l.Allow("k"); d != want {
		t.Errorf("Allow() after Refund = %+v, want %+v", d, want)
	}
}

func TestRefundAfterTimeHasPassed(t *testing.T) {
	t.Parallel()
	clk := newClock()
	l := ratelimit.New(perMinute, 10, clk.Now)
	for range perMinute.Quota {
		l.Allow("k")
	}
	clk.advance(500 * time.Millisecond)
	l.Refund("k")
	if d := l.Allow("k"); !d.Allowed || d.Remaining != 0 {
		t.Errorf("Allow() after a late Refund = %+v, want allowed with nothing left", d)
	}
}

func TestRefundNeverExceedsTheQuota(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(ratelimit.Policy{Quota: 2, Window: time.Minute}, 10, newClock().Now)
	l.Allow("k")
	l.Refund("k")
	l.Refund("k")
	want := ratelimit.Decision{Allowed: true, Remaining: 1, Reset: 30 * time.Second}
	if d := l.Allow("k"); d != want {
		t.Errorf("Allow() after two Refunds of one request = %+v, want %+v", d, want)
	}
}

func TestRefundOfAnUnknownKeyKeepsNothing(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(perMinute, 10, newClock().Now)
	l.Refund("never-seen")
	if n := ratelimit.Len(l); n != 0 {
		t.Errorf("Len() after Refund of an unknown key = %d, want 0", n)
	}
}

func TestAllowTellsAFullLimiterApart(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(ratelimit.Policy{Quota: 1, Window: time.Minute}, 1, newClock().Now)
	l.Allow("a")
	if d := l.Allow("a"); d.Allowed || d.Full {
		t.Errorf("Allow(a) over its quota = %+v, want refused and not Full", d)
	}
	if d := l.Allow("b"); d.Allowed || !d.Full {
		t.Errorf("Allow(b) on a full limiter = %+v, want refused and Full", d)
	}
}

func TestPeekSpendsNothing(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(ratelimit.Policy{Quota: 2, Window: time.Minute}, 10, newClock().Now)
	l.Allow("k")
	want := ratelimit.Decision{Allowed: true, Remaining: 1, Reset: 30 * time.Second}
	for i := range 3 {
		if d := l.Peek("k"); d != want {
			t.Fatalf("Peek() #%d = %+v, want %+v", i+1, d, want)
		}
	}
	l.Allow("k")
	if d := l.Peek("k"); d.Allowed || d.Remaining != 0 {
		t.Errorf("Peek() of a spent key = %+v, want refused with nothing left", d)
	}
}

func TestPeekOfAnUnknownKeyKeepsNothing(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(perMinute, 10, newClock().Now)
	want := ratelimit.Decision{Allowed: true, Remaining: perMinute.Quota}
	if d := l.Peek("never-seen"); d != want {
		t.Errorf("Peek() of an unknown key = %+v, want %+v", d, want)
	}
	if n := ratelimit.Len(l); n != 0 {
		t.Errorf("Len() after Peek of an unknown key = %d, want 0", n)
	}
}

func TestChargeSpendsARequest(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(ratelimit.Policy{Quota: 2, Window: time.Minute}, 10, newClock().Now)
	l.Charge("k")
	want := ratelimit.Decision{Allowed: true, Remaining: 1, Reset: 30 * time.Second}
	if d := l.Peek("k"); d != want {
		t.Errorf("Peek() after Charge = %+v, want %+v", d, want)
	}
}

func TestChargeBeyondTheQuotaIsRepaidOverTime(t *testing.T) {
	t.Parallel()
	clk := newClock()
	l := ratelimit.New(ratelimit.Policy{Quota: 2, Window: time.Minute}, 10, clk.Now)
	for range 4 {
		l.Charge("k")
	}
	want := ratelimit.Decision{Allowed: false, Remaining: 0, Reset: 90 * time.Second}
	if d := l.Peek("k"); d != want {
		t.Fatalf("Peek() owing two requests = %+v, want %+v: three to restore before one is left", d, want)
	}
	if d := l.Allow("k"); d != want {
		t.Errorf("Allow() owing two requests = %+v, want %+v", d, want)
	}
	clk.advance(time.Minute)
	if d := l.Peek("k"); d.Allowed {
		t.Errorf("Peek() a window later = %+v, want refused: the debt is not repaid yet", d)
	}
	clk.advance(30 * time.Second)
	if d := l.Allow("k"); !d.Allowed || d.Remaining != 0 {
		t.Errorf("Allow() once the debt is repaid = %+v, want allowed with nothing left", d)
	}
}

func TestSweepKeepsABucketInDebt(t *testing.T) {
	t.Parallel()
	clk := newClock()
	l := ratelimit.New(ratelimit.Policy{Quota: 2, Window: time.Minute}, 10, clk.Now)
	for range 4 {
		l.Charge("k")
	}
	clk.advance(time.Minute)
	l.Allow("other")
	if d := l.Peek("k"); d.Allowed {
		t.Errorf("Peek() after a sweep = %+v, want refused: a bucket in debt is not forgotten", d)
	}
}

func TestChargeOnAFullLimiterKeepsNothing(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(ratelimit.Policy{Quota: 1, Window: time.Minute}, 1, newClock().Now)
	l.Allow("a")
	l.Charge("b")
	if n := ratelimit.Len(l); n != 1 {
		t.Errorf("Len() after Charge on a full limiter = %d, want 1", n)
	}
	if d := l.Peek("b"); !d.Allowed {
		t.Errorf("Peek(b) after Charge on a full limiter = %+v, want allowed: nothing was charged", d)
	}
}

func TestAllowNSpendsAllOrNothing(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(perMinute, 10, newClock().Now)

	if d := l.AllowN("k", 50); !d.Allowed || d.Remaining != 10 {
		t.Fatalf("AllowN(50) = %+v, want allowed with 10 left", d)
	}
	// 11 asked, 10 left: refused, nothing spent, and Reset is the second
	// that brings the 11th back.
	if d := l.AllowN("k", 11); d.Allowed || d.Remaining != 10 || d.Reset != time.Second {
		t.Errorf("AllowN(11) = %+v, want refused with 10 left and 1 s to wait", d)
	}
	if d := l.AllowN("k", 10); !d.Allowed || d.Remaining != 0 {
		t.Errorf("AllowN(10) = %+v, want allowed with nothing left", d)
	}
	if d := l.AllowN("k", 30); d.Allowed || d.Reset != 30*time.Second {
		t.Errorf("AllowN(30) on an empty bucket = %+v, want refused for 30 s", d)
	}
}

func TestAllowNOfMoreThanTheQuotaIsAlwaysRefused(t *testing.T) {
	t.Parallel()
	l := ratelimit.New(perMinute, 10, newClock().Now)
	if d := l.AllowN("k", perMinute.Quota+1); d.Allowed || d.Remaining != perMinute.Quota {
		t.Errorf("AllowN(quota+1) = %+v, want refused, the whole quota kept", d)
	}
}
