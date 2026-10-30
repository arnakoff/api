// SPDX-License-Identifier: AGPL-3.0-only

package arcep

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

// upstream serves the numbering file at /numbering.csv and the operators
// file at /operators.csv.
func upstream(t *testing.T, numbering, operators http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/numbering.csv", numbering)
	mux.HandleFunc("/operators.csv", operators)
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func serve(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }
}

func TestFetch(t *testing.T) {
	t.Parallel()
	srv := upstream(t, serve(readFixture(t, numberingFixture)), serve(readFixture(t, operatorsFixture)))
	f := NewNumbering(httpclienttest.NewClient(t, srv, httpclient.Config{}), srv.URL+"/numbering.csv", srv.URL+"/operators.csv")
	snap, err := f.Fetch(t.Context())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snap.Source != NumberingSource || len(snap.Rows) != 34 || snap.Rows[0][4] == "" {
		t.Errorf("Fetch() = %s with %d rows, first %q", snap.Source, len(snap.Rows), snap.Rows[0])
	}
}

func TestFetchErrors(t *testing.T) {
	t.Parallel()
	html := serve([]byte("<html></html>\n"))
	tests := []struct {
		name                string
		numbering, operator http.HandlerFunc
		operatorsURL        string
		cfg                 httpclient.Config
		want                error
	}{
		{name: "numbering of another format", numbering: html, want: ErrFormat},
		{name: "operators of another format", operator: html, want: ErrFormat},
		{name: "operators on a host not allowed", operatorsURL: OperatorsURL, want: httpclient.ErrURLNotAllowed},
		{name: "file too large", cfg: httpclient.Config{MaxBodyBytes: 1 << 10}, want: httpclient.ErrBodyTooLarge},
		{name: "file too slow", numbering: httpclienttest.Stall, cfg: httpclient.Config{Timeout: 100 * time.Millisecond}, want: context.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			numbering, operators := serve(readFixture(t, numberingFixture)), serve(readFixture(t, operatorsFixture))
			if tt.numbering != nil {
				numbering = tt.numbering
			}
			if tt.operator != nil {
				operators = tt.operator
			}
			srv := upstream(t, numbering, operators)
			operatorsURL := srv.URL + "/operators.csv"
			if tt.operatorsURL != "" {
				operatorsURL = tt.operatorsURL
			}
			f := NewNumbering(httpclienttest.NewClient(t, srv, tt.cfg), srv.URL+"/numbering.csv", operatorsURL)
			if _, err := f.Fetch(t.Context()); !errors.Is(err, tt.want) {
				t.Errorf("Fetch() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSpec(t *testing.T) {
	t.Parallel()
	spec := NewNumbering(nil, "", "").Spec()
	if spec.Source != NumberingSource || len(spec.Columns) != len(numberingColumns) || spec.MinRows <= 34 {
		t.Errorf("Spec() = %+v", spec)
	}
}

func TestHosts(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{NumberingURL, OperatorsURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || len(Hosts()) != 1 || u.Host != Hosts()[0] {
			t.Errorf("%s is not served by Hosts() = %q", raw, Hosts())
		}
	}
}
