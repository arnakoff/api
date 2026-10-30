// SPDX-License-Identifier: AGPL-3.0-only

package httpclient_test

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// newServer starts a TLS test server that counts the requests it receives.
func newServer(t *testing.T, h http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// newClient returns a client that trusts srv, allows its host unless cfg
// names other hosts, and never sleeps between attempts.
func newClient(t *testing.T, srv *httptest.Server, cfg httpclient.Config) *httpclient.Client {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	cfg.RootCAs = pool
	if cfg.AllowedHosts == nil {
		cfg.AllowedHosts = []string{hostOf(t, srv.URL)}
	}
	c, err := httpclient.New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	c.SetWait(func(context.Context, time.Duration) error { return nil })
	return c
}

func hostOf(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", rawURL, err)
	}
	return u.Host
}

func TestGetReturnsBody(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("siren;nom\n"))
	})
	c := newClient(t, srv, httpclient.Config{})

	got, err := c.Get(t.Context(), srv.URL+"/liste.csv")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != "siren;nom\n" {
		t.Errorf("Get() = %q, want %q", got, "siren;nom\n")
	}
}

func TestGetRefusesURLOutsideAllowlist(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, func(http.ResponseWriter, *http.Request) {})
	c := newClient(t, srv, httpclient.Config{})
	host := hostOf(t, srv.URL)

	tests := []struct {
		name string
		url  string
	}{
		{"plain HTTP", "http://" + host + "/"},
		{"other host", "https://example.org/"},
		{"suffix of an allowed host", "https://x" + host + "/"},
		{"credentials", "https://user:secret@" + host + "/"},
		{"relative", "/liste.csv"},
		{"unparsable", "https://" + host + "/%zz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := c.Get(t.Context(), tt.url)
			if !errors.Is(err, httpclient.ErrURLNotAllowed) {
				t.Errorf("Get(%q) error = %v, want ErrURLNotAllowed", tt.url, err)
			}
		})
	}
	t.Cleanup(func() {
		if n := hits.Load(); n != 0 {
			t.Errorf("server received %d requests, want 0", n)
		}
	})
}

func TestGetFollowsRedirectOnlyToAllowedHost(t *testing.T) {
	t.Parallel()
	other, otherHits := newServer(t, func(http.ResponseWriter, *http.Request) {})
	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/forbidden":
			http.Redirect(w, r, other.URL+"/", http.StatusFound)
		case "/plain":
			// Refused before it is sent: nothing listens there.
			http.Redirect(w, r, "http://127.0.0.1/final", http.StatusFound)
		case "/allowed":
			http.Redirect(w, r, "/final", http.StatusMovedPermanently)
		default:
			_, _ = w.Write([]byte("final"))
		}
	})
	c := newClient(t, srv, httpclient.Config{})

	got, err := c.Get(t.Context(), srv.URL+"/allowed")
	if err != nil || string(got) != "final" {
		t.Errorf("Get(/allowed) = %q, %v, want %q, nil", got, err, "final")
	}
	for _, path := range []string{"/forbidden", "/plain"} {
		if _, err := c.Get(t.Context(), srv.URL+path); !errors.Is(err, httpclient.ErrURLNotAllowed) {
			t.Errorf("Get(%s) error = %v, want ErrURLNotAllowed", path, err)
		}
	}
	if n := otherHits.Load(); n != 0 {
		t.Errorf("forbidden host received %d requests, want 0", n)
	}
}

func TestGetStopsAfterTenRedirects(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})
	c := newClient(t, srv, httpclient.Config{})

	if _, err := c.Get(t.Context(), srv.URL+"/"); err == nil {
		t.Fatal("Get() error = nil, want a redirect loop error")
	}
	if n := hits.Load(); n != 10 {
		t.Errorf("server received %d requests, want 10", n)
	}
}

func TestGetLimitsBodySize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		chunked bool
		wantErr error
	}{
		{"at the limit", "12345678", false, nil},
		{"over the limit", "123456789", false, httpclient.ErrBodyTooLarge},
		{"over the limit, no Content-Length", "123456789", true, httpclient.ErrBodyTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
				if tt.chunked {
					_, _ = w.Write([]byte(tt.body[:1]))
					w.(http.Flusher).Flush()
					_, _ = w.Write([]byte(tt.body[1:]))
					return
				}
				_, _ = w.Write([]byte(tt.body))
			})
			c := newClient(t, srv, httpclient.Config{MaxBodyBytes: 8})

			got, err := c.Get(t.Context(), srv.URL+"/")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && string(got) != tt.body {
				t.Errorf("Get() = %q, want %q", got, tt.body)
			}
		})
	}
}

func TestGetTimesOut(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, httpclienttest.Stall)
	c := newClient(t, srv, httpclient.Config{Timeout: 50 * time.Millisecond})

	_, err := c.Get(t.Context(), srv.URL+"/")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Get() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestGetTimesOutAwaitingHeaders(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, httpclienttest.Stall)
	c := newClient(t, srv, httpclient.Config{ResponseHeaderTimeout: 50 * time.Millisecond})

	_, err := c.Get(t.Context(), srv.URL+"/")
	var netErr interface{ Timeout() bool }
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Errorf("Get() error = %v, want a timeout", err)
	}
}

func TestGetHonoursCancelledContext(t *testing.T) {
	t.Parallel()
	srv, hits := newServer(t, func(http.ResponseWriter, *http.Request) {})
	c := newClient(t, srv, httpclient.Config{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := c.Get(ctx, srv.URL+"/"); !errors.Is(err, context.Canceled) {
		t.Errorf("Get() error = %v, want context.Canceled", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
}

func TestGetSendsUserAgent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  string
		want string
	}{
		{"default", "", httpclient.DefaultUserAgent},
		{"configured", "arnakoff-test (+https://github.com/arnakoff/api)", "arnakoff-test (+https://github.com/arnakoff/api)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got atomic.Value
			srv, _ := newServer(t, func(_ http.ResponseWriter, r *http.Request) {
				got.Store(r.UserAgent())
			})
			c := newClient(t, srv, httpclient.Config{UserAgent: tt.cfg})

			if _, err := c.Get(t.Context(), srv.URL+"/"); err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if got.Load() != tt.want {
				t.Errorf("User-Agent = %q, want %q", got.Load(), tt.want)
			}
		})
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	t.Parallel()
	hosts := []string{"rdap.nic.fr"}
	tests := []struct {
		name string
		cfg  httpclient.Config
	}{
		{"no host", httpclient.Config{}},
		{"empty host", httpclient.Config{AllowedHosts: []string{""}}},
		{"host with scheme", httpclient.Config{AllowedHosts: []string{"https://rdap.nic.fr"}}},
		{"negative timeout", httpclient.Config{AllowedHosts: hosts, Timeout: -time.Second}},
		{"negative dial timeout", httpclient.Config{AllowedHosts: hosts, DialTimeout: -time.Second}},
		{"negative TLS timeout", httpclient.Config{AllowedHosts: hosts, TLSHandshakeTimeout: -time.Second}},
		{"negative header timeout", httpclient.Config{AllowedHosts: hosts, ResponseHeaderTimeout: -time.Second}},
		{"negative body size", httpclient.Config{AllowedHosts: hosts, MaxBodyBytes: -1}},
		{"negative attempts", httpclient.Config{AllowedHosts: hosts, Attempts: -1}},
		{"negative backoff", httpclient.Config{AllowedHosts: hosts, BaseBackoff: -time.Second}},
		{"negative max backoff", httpclient.Config{AllowedHosts: hosts, MaxBackoff: -time.Second}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := httpclient.New(tt.cfg); !errors.Is(err, httpclient.ErrInvalidConfig) {
				t.Errorf("New() error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}
