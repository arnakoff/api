// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// upstream plays the API: it answers each query, and records the queries
// it receives.
type upstream struct {
	srv     *httptest.Server
	mu      sync.Mutex
	queries []url.Values
}

func newUpstream(t *testing.T, handler func(w http.ResponseWriter, q url.Values)) *upstream {
	t.Helper()
	u := &upstream{}
	u.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+recordsPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		u.mu.Lock()
		u.queries = append(u.queries, r.URL.Query())
		u.mu.Unlock()
		handler(w, r.URL.Query())
	}))
	t.Cleanup(u.srv.Close)
	return u
}

// serve answers a query about a SIREN with its fixture, and with no notice
// otherwise.
func serve(t *testing.T, fixtures map[string]string) func(http.ResponseWriter, url.Values) {
	t.Helper()
	bodies := make(map[string][]byte, len(fixtures))
	for n, name := range fixtures {
		bodies[`registre="`+n+`"`] = readFixture(t, name)
	}
	empty := readFixture(t, "no-notice.json")
	return func(w http.ResponseWriter, q url.Values) {
		for filter, body := range bodies {
			if strings.HasPrefix(q.Get("where"), filter) {
				_, _ = w.Write(body)
				return
			}
		}
		_, _ = w.Write(empty)
	}
}

func (u *upstream) received() []url.Values {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.queries)
}

func (u *upstream) client(t *testing.T) *Client {
	t.Helper()
	c, err := New(httpclienttest.NewClient(t, u.srv, httpclient.Config{}), u.srv.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
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

func TestNotices(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{"402175558": "rectificatif.json"}))
	got, err := u.client(t).Notices(t.Context(), siren(t, "402175558"))
	if err != nil {
		t.Fatalf("Notices() error = %v", err)
	}
	want := []domain.Notice{{ID: "A202602093931", Kind: domain.NoticeCollectiveProcedure, Published: day(2026, time.October, 30)}}
	if !slices.Equal(got, want) {
		t.Errorf("Notices() = %v, want %v", got, want)
	}
}

// The query selects no field that names a person (commercant,
// listepersonnes), so the people a notice names are never fetched.
func TestNoticesSelectsNoPersonalField(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, nil))
	_, _ = u.client(t).Notices(t.Context(), siren(t, "504352360"))
	want := url.Values{
		"where": {`registre="504352360" and familleavis in ` +
			`("collective","conciliation","radiation","retablissement_professionnel")`},
		"select":   {"id,registre,dateparution,familleavis,typeavis,publicationavis,parution,numeroannonce,parutionavisprecedent"},
		"order_by": {"dateparution desc"},
		"limit":    {"100"},
	}
	got := u.received()
	if len(got) != 1 || got[0].Encode() != want.Encode() {
		t.Errorf("queries = %v, want [%v]", got, want)
	}
}

func TestNoticesCaches(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{"402175558": "rectificatif.json"}))
	c := u.client(t)
	for _, s := range []string{"402175558", "402175558", "120027016", "120027016"} {
		_, _ = c.Notices(t.Context(), siren(t, s))
	}
	if n := len(u.received()); n != 2 {
		t.Errorf("requests = %d, want 2: notices and no notice are cached", n)
	}
}

func TestNoticesReturnsACopy(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, serve(t, map[string]string{"402175558": "rectificatif.json"}))
	c := u.client(t)
	first, err := c.Notices(t.Context(), siren(t, "402175558"))
	if err != nil {
		t.Fatalf("Notices() error = %v", err)
	}
	first[0].ID = "changed"
	again, _ := c.Notices(t.Context(), siren(t, "402175558"))
	if again[0].ID != "A202602093931" {
		t.Errorf("Notices() after a change by the caller = %v, want the cached answer", again)
	}
}

func TestNoticesStatusError(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, func(w http.ResponseWriter, _ url.Values) {
		w.WriteHeader(http.StatusBadRequest)
	})
	c := u.client(t)
	for range 2 {
		_, err := c.Notices(t.Context(), siren(t, "504352360"))
		if se, ok := errors.AsType[*httpclient.StatusError](err); !ok || se.StatusCode != http.StatusBadRequest {
			t.Fatalf("Notices() error = %v, want a 400 StatusError", err)
		}
	}
	if n := len(u.received()); n != 2 {
		t.Errorf("requests = %d, want 2: a failure is not cached", n)
	}
}

func TestNoticesInvalidResponse(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, func(w http.ResponseWriter, _ url.Values) {
		_, _ = w.Write([]byte(`{"total_count":0}`))
	})
	if _, err := u.client(t).Notices(t.Context(), siren(t, "504352360")); !errors.Is(err, ErrResponse) {
		t.Errorf("Notices() error = %v, want ErrResponse", err)
	}
}

func TestNoticesRespectsRetryAfter(t *testing.T) {
	t.Parallel()
	u := newUpstream(t, func(w http.ResponseWriter, _ url.Values) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c := u.client(t)
	now := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	c.SetNow(func() time.Time { return now })

	if _, err := c.Notices(t.Context(), siren(t, "504352360")); !errors.As(err, new(*httpclient.StatusError)) {
		t.Fatalf("first Notices() error = %v, want StatusError", err)
	}
	if _, err := c.Notices(t.Context(), siren(t, "835084732")); !errors.Is(err, ErrThrottled) {
		t.Errorf("Notices() before Retry-After error = %v, want ErrThrottled", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := c.Notices(t.Context(), siren(t, "835084732")); errors.Is(err, ErrThrottled) {
		t.Errorf("Notices() after Retry-After error = %v, want a request", err)
	}
	if n := len(u.received()); n != 2 {
		t.Errorf("requests = %d, want 2", n)
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
