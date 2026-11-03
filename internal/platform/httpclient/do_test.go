// SPDX-License-Identifier: AGPL-3.0-only

package httpclient_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

func TestDoReturnsTheResponse(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.Header.Get("If-None-Match") != `"v1"` {
			t.Errorf("request = %s If-None-Match %q, want PUT with the caller's header", r.Method, r.Header.Get("If-None-Match"))
		}
		w.WriteHeader(http.StatusNotModified)
	})
	c := newClient(t, srv, httpclient.Config{})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPut, srv.URL+"/b/k", strings.NewReader("x"))
	req.Header.Set("If-None-Match", `"v1"`)

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Errorf("StatusCode = %d, want 304 as is", resp.StatusCode)
	}
}

func TestDoRefusesAHostNotAllowed(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, func(http.ResponseWriter, *http.Request) {})
	c := newClient(t, srv, httpclient.Config{AllowedHosts: []string{"example.org"}})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

	resp, err := c.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, httpclient.ErrURLNotAllowed) || hits.Load() != 0 {
		t.Errorf("Do() error = %v after %d requests, want ErrURLNotAllowed and none sent", err, hits.Load())
	}
}

func TestDoRefusesARedirectToAHostNotAllowed(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.org/", http.StatusFound)
	})
	c := newClient(t, srv, httpclient.Config{})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

	resp, err := c.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, httpclient.ErrURLNotAllowed) {
		t.Errorf("Do() error = %v, want ErrURLNotAllowed", err)
	}
}

func TestDoCapsTheBody(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		announce bool
	}{
		{"announced length", true},
		{"chunked", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
				if tt.announce {
					w.Header().Set("Content-Length", "11")
				}
				_, _ = io.WriteString(w, "hello")
				w.(http.Flusher).Flush()
				_, _ = io.WriteString(w, " world")
			})
			c := newClient(t, srv, httpclient.Config{MaxBodyBytes: 10})
			req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

			err := doAndRead(c, req)
			if !errors.Is(err, httpclient.ErrBodyTooLarge) {
				t.Errorf("Do() and read error = %v, want ErrBodyTooLarge", err)
			}
		})
	}
}

func TestDoReadsABodyAtTheLimit(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "0123456789")
	})
	c := newClient(t, srv, httpclient.Config{MaxBodyBytes: 10})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

	if err := doAndRead(c, req); err != nil {
		t.Errorf("Do() and read error = %v, want none", err)
	}
}

func TestDoBoundsTheExchange(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, httpclienttest.Stall)
	c := newClient(t, srv, httpclient.Config{Timeout: 50 * time.Millisecond})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)

	start := time.Now()
	err := doAndRead(c, req)
	if err == nil || time.Since(start) > 5*time.Second {
		t.Errorf("Do() error = %v after %v, want a timeout", err, time.Since(start))
	}
}

// doAndRead sends req with c and reads the whole response body.
func doAndRead(c *httpclient.Client, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, err = io.ReadAll(resp.Body)
	return err
}
