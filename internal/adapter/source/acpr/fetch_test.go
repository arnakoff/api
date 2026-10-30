// SPDX-License-Identifier: AGPL-3.0-only

package acpr

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

func upstream(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func serve(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }
}

func TestFetch(t *testing.T) {
	t.Parallel()
	srv := upstream(t, serve(readFixture(t, csvFixture)))
	f := NewBlacklist(httpclienttest.NewClient(t, srv, httpclient.Config{}), srv.URL+"/liste.csv")
	snap, err := f.Fetch(t.Context())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snap.Source != BlacklistSource || len(snap.Rows) != 29 {
		t.Errorf("Fetch() = %s with %d rows", snap.Source, len(snap.Rows))
	}
}

func TestFetchErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		handler http.HandlerFunc
		fileURL string
		cfg     httpclient.Config
		want    error
	}{
		{name: "file of another format", handler: serve([]byte("<html></html>\n")), want: ErrFormat},
		{name: "host not allowed", fileURL: BlacklistURL, want: httpclient.ErrURLNotAllowed},
		{name: "file too large", cfg: httpclient.Config{MaxBodyBytes: 1 << 10}, want: httpclient.ErrBodyTooLarge},
		{name: "file too slow", handler: httpclienttest.Stall, cfg: httpclient.Config{Timeout: 100 * time.Millisecond}, want: context.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := serve(readFixture(t, csvFixture))
			if tt.handler != nil {
				h = tt.handler
			}
			srv := upstream(t, h)
			fileURL := srv.URL + "/liste.csv"
			if tt.fileURL != "" {
				fileURL = tt.fileURL
			}
			f := NewBlacklist(httpclienttest.NewClient(t, srv, tt.cfg), fileURL)
			if _, err := f.Fetch(t.Context()); !errors.Is(err, tt.want) {
				t.Errorf("Fetch() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSpec(t *testing.T) {
	t.Parallel()
	spec := NewBlacklist(nil, "").Spec()
	if spec.Source != BlacklistSource || len(spec.Columns) != len(blacklistColumns) || spec.MinRows <= 29 {
		t.Errorf("Spec() = %+v", spec)
	}
}

func TestHosts(t *testing.T) {
	t.Parallel()
	u, err := url.Parse(BlacklistURL)
	if err != nil || u.Scheme != "https" || len(Hosts()) != 1 || u.Host != Hosts()[0] {
		t.Errorf("%s is not served by Hosts() = %q", BlacklistURL, Hosts())
	}
}
