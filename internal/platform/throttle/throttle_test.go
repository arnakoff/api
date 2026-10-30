// SPDX-License-Identifier: AGPL-3.0-only

package throttle_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
	"github.com/arnakoff/api/internal/platform/throttle"
)

const host = "rdap.nic.fr"

// fakeClock is a settable clock.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func newClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)}
}

func TestAllowsUntilTheWaitIsOver(t *testing.T) {
	t.Parallel()
	clk := newClock()
	th := throttle.New(clk.Now)
	if !th.Allows(host) {
		t.Fatal("Allows() on a new host = false, want true")
	}
	th.Wait(host, 2*time.Minute)
	if th.Allows(host) {
		t.Error("Allows() during the wait = true, want false")
	}
	if !th.Allows("rdap.verisign.com") {
		t.Error("Allows() on another host = false, want true")
	}
	clk.now = clk.now.Add(2 * time.Minute)
	if !th.Allows(host) {
		t.Error("Allows() once the wait is over = false, want true")
	}
}

func TestWaitIsCapped(t *testing.T) {
	t.Parallel()
	clk := newClock()
	th := throttle.New(clk.Now)
	th.Wait(host, 365*24*time.Hour)
	clk.now = clk.now.Add(throttle.MaxWait)
	if !th.Allows(host) {
		t.Errorf("Allows() after %v = false, want true", throttle.MaxWait)
	}
}

func TestHeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		retryAfter string
		status     int
		wantWait   bool
	}{
		{"429 with Retry-After", "120", http.StatusTooManyRequests, true},
		{"503 with Retry-After", "120", http.StatusServiceUnavailable, true},
		{"429 without Retry-After", "", http.StatusTooManyRequests, false},
		{"404", "", http.StatusNotFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := statusError(t, tt.status, tt.retryAfter)
			clk := newClock()
			th := throttle.New(clk.Now)
			th.Heed(host, err)
			if got := !th.Allows(host); got != tt.wantWait {
				t.Errorf("waiting after Heed(%v) = %v, want %v", err, got, tt.wantWait)
			}
		})
	}
}

func TestHeedIgnoresOtherErrors(t *testing.T) {
	t.Parallel()
	th := throttle.New(newClock().Now)
	th.Heed(host, fmt.Errorf("lookup: %w", errors.New("connection refused")))
	th.Heed(host, nil)
	if !th.Allows(host) {
		t.Error("Allows() after a non-HTTP error = false, want true")
	}
}

// statusError returns the error an httpclient.Client gives for an answer
// with status and, if not empty, a Retry-After header.
func statusError(t *testing.T, status int, retryAfter string) error {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	c := httpclienttest.NewClient(t, srv, httpclient.Config{})
	_, err := c.Get(t.Context(), srv.URL)
	if _, ok := errors.AsType[*httpclient.StatusError](err); !ok {
		t.Fatalf("Get() error = %v, want a StatusError", err)
	}
	return fmt.Errorf("query: %w", err)
}
