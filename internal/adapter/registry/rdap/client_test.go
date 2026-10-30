// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// upstream plays IANA at /dns.json and the registries of the bootstrap
// fixture: .fr on one test server, .com on another, so they are two hosts.
type upstream struct {
	fr, com   *httptest.Server
	mu        sync.Mutex
	paths     []string
	bootstrap atomic.Int32
	dialed    [][]string
}

func newUpstream(t *testing.T, registry http.HandlerFunc) *upstream {
	t.Helper()
	u := &upstream{}
	mux := http.NewServeMux()
	u.fr = httptest.NewTLSServer(mux)
	t.Cleanup(u.fr.Close)
	u.com = httptest.NewTLSServer(mux)
	t.Cleanup(u.com.Close)
	dns := string(readFixture(t, "iana-dns.json"))
	dns = strings.ReplaceAll(dns, "https://rdap.nic.fr/", u.fr.URL+"/fr/")
	dns = strings.ReplaceAll(dns, "https://rdap.verisign.com/com/v1/", u.com.URL+"/com/v1/")
	mux.HandleFunc("/dns.json", func(w http.ResponseWriter, _ *http.Request) {
		u.bootstrap.Add(1)
		_, _ = w.Write([]byte(dns))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u.mu.Lock()
		u.paths = append(u.paths, r.URL.Path)
		u.mu.Unlock()
		registry(w, r)
	})
	return u
}

func (u *upstream) requests() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.paths)
}

// client returns a Client of u. Its registry client allows exactly the
// hosts it is built for, which it records.
func (u *upstream) client(t *testing.T) *Client {
	t.Helper()
	iana := httpclienttest.NewClient(t, u.fr, httpclient.Config{})
	c, err := New(iana, u.fr.URL+"/dns.json", u.dial(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

func (u *upstream) dial(t *testing.T) func([]string) (Getter, error) {
	t.Helper()
	// Every httptest TLS server serves the same certificate.
	pool := x509.NewCertPool()
	pool.AddCert(u.fr.Certificate())
	return func(hosts []string) (Getter, error) {
		u.mu.Lock()
		u.dialed = append(u.dialed, hosts)
		u.mu.Unlock()
		return httpclient.New(httpclient.Config{AllowedHosts: hosts, RootCAs: pool, Attempts: 1})
	}
}

func hostOf(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "https://")
}

// serve answers each registry path with a fixture, and 404 otherwise.
func serve(t *testing.T, fixtures map[string]string) http.HandlerFunc {
	t.Helper()
	bodies := make(map[string][]byte, len(fixtures))
	for path, name := range fixtures {
		bodies[path] = readFixture(t, name)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(readFixture(t, "afnic-not-found.json"))
			return
		}
		_, _ = w.Write(body)
	}
}

func parse(t *testing.T, s string) domain.Domain {
	t.Helper()
	d, err := domain.ParseDomain(s)
	if err != nil {
		t.Fatalf("ParseDomain(%q) error = %v", s, err)
	}
	return d
}

func TestLookup(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{
		"/fr/domain/afnic.fr":         "afnic-afnic.fr.json",
		"/com/v1/domain/verisign.com": "verisign-verisign.com.json",
	}))
	c := u.client(t)
	tests := []struct {
		in      string
		want    time.Time
		wantURL string
	}{
		{"https://www.afnic.fr/fr/", date(t, "2001-12-10T23:00:00Z"), u.fr.URL + "/fr/domain/afnic.fr"},
		{"verisign.com", date(t, "1995-06-02T04:00:00Z"), u.com.URL + "/com/v1/domain/verisign.com"},
	}
	for _, tt := range tests {
		got, err := c.Lookup(t.Context(), parse(t, tt.in))
		if err != nil {
			t.Fatalf("Lookup(%q) error = %v", tt.in, err)
		}
		if !got.Registered.Equal(tt.want) {
			t.Errorf("Lookup(%q).Registered = %v, want %v", tt.in, got.Registered, tt.want)
		}
		if got.URL != tt.wantURL {
			t.Errorf("Lookup(%q).URL = %q, want %q", tt.in, got.URL, tt.wantURL)
		}
	}
	want := []string{"/fr/domain/afnic.fr", "/com/v1/domain/verisign.com"}
	if got := u.requests(); !slices.Equal(got, want) {
		t.Errorf("registry requests = %v, want %v", got, want)
	}
}

func TestLookupAllowsTheBootstrapHosts(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{"/fr/domain/afnic.fr": "afnic-afnic.fr.json"}))
	if _, err := u.client(t).Lookup(t.Context(), parse(t, "afnic.fr")); err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	want := []string{"rdap.centralnic.com", "rdap.nic.re", "rdap.verisign.com", hostOf(u.fr), hostOf(u.com)}
	slices.Sort(want)
	if len(u.dialed) != 1 || !slices.Equal(u.dialed[0], want) {
		t.Errorf("registry client built for %v, want once for %v", u.dialed, want)
	}
}

func TestLookupCaches(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{"/fr/domain/afnic.fr": "afnic-afnic.fr.json"}))
	c := u.client(t)
	for _, in := range []string{"afnic.fr", "www.afnic.fr", "absent.fr", "absent.fr"} {
		_, _ = c.Lookup(t.Context(), parse(t, in))
	}
	want := []string{"/fr/domain/afnic.fr", "/fr/domain/absent.fr"}
	if got := u.requests(); !slices.Equal(got, want) {
		t.Errorf("registry requests = %v, want %v", got, want)
	}
	if n := u.bootstrap.Load(); n != 1 {
		t.Errorf("bootstrap fetched %d times, want 1", n)
	}
}

func TestLookupErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		in       string
		registry http.HandlerFunc
		want     error
	}{
		{"not found", "absent.fr", serve(t, nil), ErrNotFound},
		{"no server for the extension", "exemple.io", serve(t, nil), ErrNoServer},
		{"not a domain object", "afnic.fr", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"objectClassName":"entity"}`))
		}, ErrResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := newUpstream(t, tt.registry)
			if _, err := u.client(t).Lookup(t.Context(), parse(t, tt.in)); !errors.Is(err, tt.want) {
				t.Errorf("Lookup(%q) error = %v, want %v", tt.in, err, tt.want)
			}
		})
	}
}

func TestLookupBootstrapFailure(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	var hits atomic.Int32
	mux.HandleFunc("/dns.json", func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"version":"2.0"}`))
	})
	iana := httpclienttest.NewClient(t, srv, httpclient.Config{})
	c, err := New(iana, srv.URL+"/dns.json", func([]string) (Getter, error) {
		t.Error("registry client built without a bootstrap file")
		return nil, errors.New("unreachable")
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	for range 2 {
		if _, err := c.Lookup(t.Context(), parse(t, "afnic.fr")); !errors.Is(err, ErrBootstrap) {
			t.Errorf("Lookup() error = %v, want ErrBootstrap", err)
		}
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("bootstrap fetched %d times, want 1 within the negative TTL", n)
	}
}

func TestLookupRespectsRetryAfter(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c := u.client(t)
	now := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	c.SetNow(func() time.Time { return now })

	var se *httpclient.StatusError
	if _, err := c.Lookup(t.Context(), parse(t, "un.fr")); !errors.As(err, &se) {
		t.Fatalf("first Lookup() error = %v, want StatusError", err)
	}
	if _, err := c.Lookup(t.Context(), parse(t, "deux.fr")); !errors.Is(err, ErrThrottled) {
		t.Errorf("Lookup() before Retry-After error = %v, want ErrThrottled", err)
	}
	if _, err := c.Lookup(t.Context(), parse(t, "verisign.com")); errors.Is(err, ErrThrottled) {
		t.Errorf("Lookup() on another registry error = %v, want a request", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := c.Lookup(t.Context(), parse(t, "deux.fr")); errors.Is(err, ErrThrottled) {
		t.Errorf("Lookup() after Retry-After error = %v, want a request", err)
	}
	want := []string{"/fr/domain/un.fr", "/com/v1/domain/verisign.com", "/fr/domain/deux.fr"}
	if got := u.requests(); !slices.Equal(got, want) {
		t.Errorf("registry requests = %v, want %v", got, want)
	}
}

func TestBootstrapHosts(t *testing.T) {
	t.Parallel()
	u, err := url.Parse(BootstrapURL)
	if err != nil {
		t.Fatal(err)
	}
	if got := BootstrapHosts(); !slices.Equal(got, []string{u.Host}) {
		t.Errorf("BootstrapHosts() = %v, want [%s]", got, u.Host)
	}
}
