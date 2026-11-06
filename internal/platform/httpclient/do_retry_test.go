// SPDX-License-Identifier: AGPL-3.0-only

package httpclient_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
)

func TestDoRetriesAnIdempotentRequestWhenAsked(t *testing.T) {
	t.Parallel()
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var bodies []string
			next := statuses(nil, http.StatusServiceUnavailable, http.StatusTooManyRequests)
			srv, hits := newServer(t, func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				mu.Lock()
				bodies = append(bodies, string(b))
				mu.Unlock()
				next(w, r)
			})
			c := newClient(t, srv, httpclient.Config{RetryDo: true})
			req, _ := http.NewRequestWithContext(t.Context(), method, srv.URL+"/k", strings.NewReader(`{"v":1}`))

			status := doStatus(t, c, req)
			if status != http.StatusOK || hits.Load() != 3 {
				t.Errorf("Do() = %d after %d requests, want 200 after 3", status, hits.Load())
			}
			if want := []string{`{"v":1}`, `{"v":1}`, `{"v":1}`}; !slices.Equal(bodies, want) {
				t.Errorf("bodies sent = %q, want %q", bodies, want)
			}
		})
	}
}

func TestDoSendsOnceWithoutRetry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		retryDo bool
		method  string
		body    io.Reader
	}{
		{"retries not asked", false, http.MethodGet, nil},
		{"POST", true, http.MethodPost, strings.NewReader("x")},
		{"PATCH", true, http.MethodPatch, strings.NewReader("x")},
		{"body that cannot be replayed", true, http.MethodPut, io.NopCloser(strings.NewReader("x"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, hits := newServer(t, statuses(nil, http.StatusServiceUnavailable))
			c := newClient(t, srv, httpclient.Config{RetryDo: tt.retryDo})
			req, _ := http.NewRequestWithContext(t.Context(), tt.method, srv.URL, tt.body)

			status := doStatus(t, c, req)
			if status != http.StatusServiceUnavailable || hits.Load() != 1 {
				t.Errorf("Do() = %d after %d requests, want the 503 after 1", status, hits.Load())
			}
		})
	}
}

func TestDoReturnsTheLastAnswerOnceRetriesEnd(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		retryAfter string
		codes      []int
		want       int
		hits       int32
	}{
		{"attempts spent", "", []int{502, 503, 504}, 504, 3},
		{"final status", "", []int{503, 404}, 404, 2},
		{"Retry-After past the max backoff", "120", []int{429}, 429, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			header := http.Header{}
			if tt.retryAfter != "" {
				header.Set("Retry-After", tt.retryAfter)
			}
			srv, hits := newServer(t, statuses(header, tt.codes...))
			c := newClient(t, srv, httpclient.Config{RetryDo: true, MaxBackoff: time.Minute})
			req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

			status := doStatus(t, c, req)
			if status != tt.want || hits.Load() != tt.hits {
				t.Errorf("Do() = %d after %d requests, want %d after %d", status, hits.Load(), tt.want, tt.hits)
			}
		})
	}
}

func TestDoHonoursRetryAfter(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, statuses(http.Header{"Retry-After": {"3"}}, http.StatusTooManyRequests))
	c := newClient(t, srv, httpclient.Config{RetryDo: true})
	var w waits
	c.SetWait(w.record)
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

	if status := doStatus(t, c, req); status != http.StatusOK {
		t.Fatalf("Do() = %d, want 200", status)
	}
	if ds := w.all(); !slices.Equal(ds, []time.Duration{3 * time.Second}) {
		t.Errorf("waits = %v, want [3s]", ds)
	}
}

func TestDoStopsWhenTheWaitFails(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, statuses(nil, http.StatusServiceUnavailable))
	c := newClient(t, srv, httpclient.Config{RetryDo: true})
	c.SetWait(func(context.Context, time.Duration) error { return context.Canceled })
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

	resp, err := c.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, context.Canceled) || hits.Load() != 1 {
		t.Errorf("Do() error = %v after %d requests, want context.Canceled after 1", err, hits.Load())
	}
}

// doStatus sends req with c, reads and closes the body, and returns the
// status of the answer.
func doStatus(t *testing.T, c *httpclient.Client, req *http.Request) int {
	t.Helper()
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode
}
