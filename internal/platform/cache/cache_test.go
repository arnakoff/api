// SPDX-License-Identifier: AGPL-3.0-only

package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/arnakoff/api/internal/platform/cache"
)

var (
	errNotFound = errors.New("not found")
	errUpstream = errors.New("upstream down")
)

// clock is a settable time source.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *clock {
	return &clock{now: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// loader counts its calls and answers with value or err.
type loader struct {
	calls atomic.Int64
	value string
	err   error
}

func (l *loader) load(context.Context) (string, error) {
	l.calls.Add(1)
	return l.value, l.err
}

func newCache(t *testing.T, cfg cache.Config) (*cache.Cache[string], *clock) {
	t.Helper()
	c, err := cache.New[string](cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	clk := newClock()
	c.SetNow(clk.Now)
	return c, clk
}

func isNotFound(err error) bool { return errors.Is(err, errNotFound) }

func TestGetReturnsLoadedValueAndCachesIt(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t, cache.Config{})
	l := &loader{value: "v"}

	for range 3 {
		got, err := c.Get(t.Context(), "k", l.load)
		if err != nil || got != "v" {
			t.Fatalf("Get = %q, %v; want %q, nil", got, err, "v")
		}
	}
	if n := l.calls.Load(); n != 1 {
		t.Errorf("load called %d times, want 1", n)
	}
}

func TestGetKeepsKeysApart(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t, cache.Config{})
	a, b := &loader{value: "a"}, &loader{value: "b"}

	gotA, _ := c.Get(t.Context(), "a", a.load)
	gotB, _ := c.Get(t.Context(), "b", b.load)
	if gotA != "a" || gotB != "b" {
		t.Errorf("Get = %q, %q; want a, b", gotA, gotB)
	}
}

func TestGetReloadsAfterExpiry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       error
		ttl       time.Duration
		wantCalls int64
	}{
		{"value fresh", nil, time.Hour - time.Nanosecond, 1},
		{"value expired", nil, time.Hour, 2},
		{"negative fresh", errNotFound, time.Minute - time.Nanosecond, 1},
		{"negative expired", errNotFound, time.Minute, 2},
		{"other error never cached", errUpstream, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, clk := newCache(t, cache.Config{TTL: time.Hour, NegativeTTL: time.Minute, Negative: isNotFound})
			l := &loader{value: "v", err: tt.err}

			_, _ = c.Get(t.Context(), "k", l.load)
			clk.advance(tt.ttl)
			_, err := c.Get(t.Context(), "k", l.load)

			if !errors.Is(err, tt.err) {
				t.Errorf("err = %v, want %v", err, tt.err)
			}
			if n := l.calls.Load(); n != tt.wantCalls {
				t.Errorf("load called %d times, want %d", n, tt.wantCalls)
			}
		})
	}
}

func TestGetNeverCachesErrorsWithoutNegative(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t, cache.Config{})
	l := &loader{err: errNotFound}

	_, _ = c.Get(t.Context(), "k", l.load)
	_, _ = c.Get(t.Context(), "k", l.load)
	if n := l.calls.Load(); n != 2 {
		t.Errorf("load called %d times, want 2", n)
	}
}

func TestGetEvictsLeastRecentlyUsed(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t, cache.Config{MaxEntries: 2})
	ls := map[string]*loader{"a": {value: "a"}, "b": {value: "b"}, "c": {value: "c"}}

	for _, k := range []string{"a", "b", "a", "c"} {
		_, _ = c.Get(t.Context(), k, ls[k].load)
	}
	if n := c.Len(); n != 2 {
		t.Errorf("Len = %d, want 2", n)
	}
	_, _ = c.Get(t.Context(), "a", ls["a"].load)
	_, _ = c.Get(t.Context(), "b", ls["b"].load)
	if n := ls["a"].calls.Load(); n != 1 {
		t.Errorf("a loaded %d times, want 1 (recently used, kept)", n)
	}
	if n := ls["b"].calls.Load(); n != 2 {
		t.Errorf("b loaded %d times, want 2 (least recently used, evicted)", n)
	}
}

func TestGetReplacesExpiredEntryInPlace(t *testing.T) {
	t.Parallel()
	c, clk := newCache(t, cache.Config{MaxEntries: 2, TTL: time.Minute})
	l := &loader{value: "v"}

	_, _ = c.Get(t.Context(), "k", l.load)
	clk.advance(time.Minute)
	_, _ = c.Get(t.Context(), "k", l.load)
	if n := c.Len(); n != 1 {
		t.Errorf("Len = %d, want 1", n)
	}
}

func TestGetDeduplicatesConcurrentLoads(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c, err := cache.New[string](cache.Config{})
		if err != nil {
			t.Fatal(err)
		}
		release := make(chan struct{})
		var calls atomic.Int64
		load := func(context.Context) (string, error) {
			calls.Add(1)
			<-release
			return "v", nil
		}

		const callers = 8
		results := make(chan string, callers)
		for range callers {
			go func() {
				v, _ := c.Get(t.Context(), "k", load)
				results <- v
			}()
		}
		synctest.Wait() // every caller is now waiting on the one load
		close(release)
		for range callers {
			if v := <-results; v != "v" {
				t.Errorf("Get = %q, want v", v)
			}
		}
		if n := calls.Load(); n != 1 {
			t.Errorf("load called %d times, want 1", n)
		}
	})
}

func TestGetReturnsWhenCallerGivesUp(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c, err := cache.New[string](cache.Config{})
		if err != nil {
			t.Fatal(err)
		}
		release := make(chan struct{})
		loadCtxErr := make(chan error, 1)
		load := func(ctx context.Context) (string, error) {
			<-release
			loadCtxErr <- ctx.Err()
			return "v", nil
		}

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() {
			_, err := c.Get(ctx, "k", load)
			done <- err
		}()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("Get err = %v, want context.Canceled", err)
		}

		close(release)
		if err := <-loadCtxErr; err != nil {
			t.Errorf("load context err = %v, want nil: one caller leaving must not cancel the shared load", err)
		}
		synctest.Wait() // the load has returned and its value is stored
		got, err := c.Get(t.Context(), "k", func(context.Context) (string, error) {
			return "", errUpstream
		})
		if err != nil || got != "v" {
			t.Errorf("Get after load = %q, %v; want v, nil (cached)", got, err)
		}
	})
}

func TestNewRejectsNegativeConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  cache.Config
	}{
		{"max entries", cache.Config{MaxEntries: -1}},
		{"ttl", cache.Config{TTL: -time.Second}},
		{"negative ttl", cache.Config{NegativeTTL: -time.Second}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := cache.New[string](tt.cfg); !errors.Is(err, cache.ErrInvalidConfig) {
				t.Errorf("New err = %v, want ErrInvalidConfig", err)
			}
		})
	}
}
