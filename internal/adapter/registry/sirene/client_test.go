// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// upstream plays the API: it answers each q with a fixture, and records
// the queries it receives.
type upstream struct {
	srv     *httptest.Server
	mu      sync.Mutex
	queries []url.Values
}

func newUpstream(t *testing.T, handler func(w http.ResponseWriter, q string)) *upstream {
	t.Helper()
	u := &upstream{}
	u.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		u.mu.Lock()
		u.queries = append(u.queries, r.URL.Query())
		u.mu.Unlock()
		handler(w, r.URL.Query().Get("q"))
	}))
	t.Cleanup(u.srv.Close)
	return u
}

// serve answers each q with a fixture, and with no result otherwise.
func serve(t *testing.T, fixtures map[string]string) func(http.ResponseWriter, string) {
	t.Helper()
	bodies := make(map[string][]byte, len(fixtures))
	for q, name := range fixtures {
		bodies[q] = readFixture(t, name)
	}
	empty := readFixture(t, "no-result.json")
	return func(w http.ResponseWriter, q string) {
		body, ok := bodies[q]
		if !ok {
			body = empty
		}
		_, _ = w.Write(body)
	}
}

func (u *upstream) received() []url.Values {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.queries)
}

// client returns a Client of u with no request rate limit.
func (u *upstream) client(t *testing.T) *Client {
	t.Helper()
	c, err := New(httpclienttest.NewClient(t, u.srv, httpclient.Config{}), u.srv.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	c.SetLimiter(rate.NewLimiter(rate.Inf, 1))
	return c
}

func siren(t *testing.T, s string) domain.SIREN {
	t.Helper()
	n, err := domain.ParseSIREN(s)
	if err != nil {
		t.Fatalf("ParseSIREN(%q) error = %v", s, err)
	}
	return n
}

func TestLookup(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{"120027016": "insee-active.json"}))
	got, err := u.client(t).Lookup(t.Context(), siren(t, "120027016"))
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if want := "INSTITUT NATIONAL DE LA STATISTIQUE ET DES ETUDES ECONOMIQUES (INSEE)"; got.Name != want {
		t.Errorf("Lookup().Name = %q, want %q", got.Name, want)
	}
}

// The query asks for the head office only: the minimal answer leaves out
// the dirigeants, who are people, so they are never fetched.
func TestLookupAsksForTheHeadOfficeOnly(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, nil))
	_, _ = u.client(t).Lookup(t.Context(), siren(t, "120027016"))
	want := url.Values{
		"q":        {"120027016"},
		"minimal":  {"true"},
		"include":  {"siege"},
		"per_page": {"5"},
	}
	got := u.received()
	if len(got) != 1 || got[0].Encode() != want.Encode() {
		t.Errorf("queries = %v, want [%v]", got, want)
	}
}

func TestLookupCaches(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{"120027016": "insee-active.json"}))
	c := u.client(t)
	for _, s := range []string{"120027016", "120027016", "000000000", "000000000"} {
		_, _ = c.Lookup(t.Context(), siren(t, s))
	}
	if n := len(u.received()); n != 2 {
		t.Errorf("requests = %d, want 2: an answer and a not found are cached", n)
	}
}

func TestLookupNotFound(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, nil))
	if _, err := u.client(t).Lookup(t.Context(), siren(t, "000000000")); !errors.Is(err, ErrNotFound) {
		t.Errorf("Lookup() error = %v, want ErrNotFound", err)
	}
}

func TestLookupStatusError(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, func(w http.ResponseWriter, _ string) {
		w.WriteHeader(http.StatusBadRequest)
	})
	c := u.client(t)
	for range 2 {
		_, err := c.Lookup(t.Context(), siren(t, "120027016"))
		if se, ok := errors.AsType[*httpclient.StatusError](err); !ok || se.StatusCode != http.StatusBadRequest {
			t.Fatalf("Lookup() error = %v, want a 400 StatusError", err)
		}
	}
	if n := len(u.received()); n != 2 {
		t.Errorf("requests = %d, want 2: a failure is not cached", n)
	}
}

func TestLookupRespectsRetryAfter(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, func(w http.ResponseWriter, _ string) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c := u.client(t)
	now := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	c.SetNow(func() time.Time { return now })

	if _, err := c.Lookup(t.Context(), siren(t, "120027016")); !errors.As(err, new(*httpclient.StatusError)) {
		t.Fatalf("first Lookup() error = %v, want StatusError", err)
	}
	if _, err := c.Lookup(t.Context(), siren(t, "552032534")); !errors.Is(err, ErrThrottled) {
		t.Errorf("Lookup() before Retry-After error = %v, want ErrThrottled", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := c.Lookup(t.Context(), siren(t, "552032534")); errors.Is(err, ErrThrottled) {
		t.Errorf("Lookup() after Retry-After error = %v, want a request", err)
	}
	if n := len(u.received()); n != 2 {
		t.Errorf("requests = %d, want 2", n)
	}
}

func TestLookupKeepsToTheRequestRate(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, nil))
	c := u.client(t)
	c.SetLimiter(rate.NewLimiter(rate.Every(time.Hour), 1))
	if _, err := c.Lookup(t.Context(), siren(t, "000000000")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("first Lookup() error = %v, want ErrNotFound", err)
	}
	// The wait for the next request would exceed maxRateWait.
	if _, err := c.Lookup(t.Context(), siren(t, "120027016")); !errors.Is(err, ErrThrottled) {
		t.Errorf("Lookup() over the rate error = %v, want ErrThrottled", err)
	}
	if n := len(u.received()); n != 1 {
		t.Errorf("requests = %d, want 1", n)
	}
}

func TestDefaultRequestRate(t *testing.T) {
	t.Parallel()
	c, err := New(nil, BaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if l := c.Limiter(); l.Limit() != MaxRequestsPerSecond || l.Burst() != 1 {
		t.Errorf("limiter = %v per second, burst %d; want %v, burst 1", l.Limit(), l.Burst(), MaxRequestsPerSecond)
	}
}

func TestHosts(t *testing.T) {
	t.Parallel()
	u, err := url.Parse(BaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if got := Hosts(); !slices.Equal(got, []string{u.Host}) {
		t.Errorf("Hosts() = %v, want [%s]", got, u.Host)
	}
}
