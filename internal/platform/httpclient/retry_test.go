// SPDX-License-Identifier: AGPL-3.0-only

package httpclient_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
)

// waits records the pauses a client asks for instead of sleeping.
type waits struct {
	mu sync.Mutex
	ds []time.Duration
}

func (w *waits) record(_ context.Context, d time.Duration) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ds = append(w.ds, d)
	return nil
}

func (w *waits) all() []time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.ds)
}

// statuses answers each request with the next status, then 200.
func statuses(header http.Header, codes ...int) http.HandlerFunc {
	var mu sync.Mutex
	return func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if len(codes) == 0 {
			_, _ = w.Write([]byte("ok"))
			return
		}
		for k, v := range header {
			w.Header()[k] = v
		}
		w.WriteHeader(codes[0])
		codes = codes[1:]
	}
}

func TestGetHonoursRetryAfter(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		retryAfter string
		want       time.Duration
	}{
		{"seconds", "3", 3 * time.Second},
		{"zero seconds", "0", 0},
		{"HTTP date", now.Add(5 * time.Second).Format(http.TimeFormat), 5 * time.Second},
		{"past HTTP date", now.Add(-time.Hour).Format(http.TimeFormat), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, hits := newServer(t, statuses(http.Header{"Retry-After": {tt.retryAfter}}, http.StatusTooManyRequests))
			c := newClient(t, srv, httpclient.Config{MaxBackoff: time.Minute})
			c.SetNow(func() time.Time { return now })
			var w waits
			c.SetWait(w.record)

			got, err := c.Get(t.Context(), srv.URL+"/")
			if err != nil || string(got) != "ok" {
				t.Fatalf("Get() = %q, %v, want %q, nil", got, err, "ok")
			}
			if ds := w.all(); !slices.Equal(ds, []time.Duration{tt.want}) {
				t.Errorf("waits = %v, want [%v]", ds, tt.want)
			}
			if n := hits.Load(); n != 2 {
				t.Errorf("server received %d requests, want 2", n)
			}
		})
	}
}

func TestGetGivesUpWhenRetryAfterExceedsMaxBackoff(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, statuses(http.Header{"Retry-After": {"3600"}}, http.StatusTooManyRequests))
	c := newClient(t, srv, httpclient.Config{MaxBackoff: time.Minute})
	var w waits
	c.SetWait(w.record)

	_, err := c.Get(t.Context(), srv.URL+"/")
	var se *httpclient.StatusError
	if !errors.As(err, &se) || se.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("Get() error = %v, want StatusError 429", err)
	}
	if d, ok := se.RetryAfter(); !ok || d != time.Hour {
		t.Errorf("RetryAfter() = %v, %t, want %v, true", d, ok, time.Hour)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("server received %d requests, want 1", n)
	}
	if ds := w.all(); len(ds) != 0 {
		t.Errorf("waits = %v, want none", ds)
	}
}

func TestGetRetriesServerErrorsWithBackoff(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, statuses(http.Header{"Retry-After": {"soon"}}, http.StatusServiceUnavailable, http.StatusInternalServerError))
	base, maxBackoff := 100*time.Millisecond, 150*time.Millisecond
	c := newClient(t, srv, httpclient.Config{Attempts: 3, BaseBackoff: base, MaxBackoff: maxBackoff})
	var w waits
	c.SetWait(w.record)

	got, err := c.Get(t.Context(), srv.URL+"/")
	if err != nil || string(got) != "ok" {
		t.Fatalf("Get() = %q, %v, want %q, nil", got, err, "ok")
	}
	if n := hits.Load(); n != 3 {
		t.Errorf("server received %d requests, want 3", n)
	}
	ds := w.all()
	if len(ds) != 2 {
		t.Fatalf("waits = %v, want 2", ds)
	}
	// Full jitter: the n-th pause is drawn from [0, min(base·2ⁿ, max)].
	for i, ceiling := range []time.Duration{base, maxBackoff} {
		if ds[i] < 0 || ds[i] > ceiling {
			t.Errorf("wait %d = %v, want within [0, %v]", i, ds[i], ceiling)
		}
	}
}

func TestGetStopsAfterLastAttempt(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, statuses(nil, http.StatusBadGateway, http.StatusBadGateway, http.StatusBadGateway))
	c := newClient(t, srv, httpclient.Config{Attempts: 3})

	_, err := c.Get(t.Context(), srv.URL+"/")
	var se *httpclient.StatusError
	if !errors.As(err, &se) || se.StatusCode != http.StatusBadGateway {
		t.Fatalf("Get() error = %v, want StatusError 502", err)
	}
	if n := hits.Load(); n != 3 {
		t.Errorf("server received %d requests, want 3", n)
	}
}

func TestStatusErrorWithoutRetryAfter(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, statuses(nil, http.StatusNotFound))
	_, err := newClient(t, srv, httpclient.Config{}).Get(t.Context(), srv.URL+"/")
	var se *httpclient.StatusError
	if !errors.As(err, &se) {
		t.Fatalf("Get() error = %v, want StatusError", err)
	}
	if d, ok := se.RetryAfter(); ok {
		t.Errorf("RetryAfter() = %v, true, want false", d)
	}
}

func TestGetDoesNotRetryClientErrors(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, statuses(nil, http.StatusNotFound))
	c := newClient(t, srv, httpclient.Config{Attempts: 3})

	_, err := c.Get(t.Context(), srv.URL+"/")
	var se *httpclient.StatusError
	if !errors.As(err, &se) || se.StatusCode != http.StatusNotFound {
		t.Fatalf("Get() error = %v, want StatusError 404", err)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("server received %d requests, want 1", n)
	}
}

func TestGetStopsWhenWaitIsCancelled(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, statuses(nil, http.StatusServiceUnavailable))
	c := newClient(t, srv, httpclient.Config{Attempts: 3})
	c.SetWait(func(context.Context, time.Duration) error { return context.Canceled })

	if _, err := c.Get(t.Context(), srv.URL+"/"); !errors.Is(err, context.Canceled) {
		t.Errorf("Get() error = %v, want context.Canceled", err)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("server received %d requests, want 1", n)
	}
}
