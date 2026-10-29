// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
)

// realFileURL is the file URL in the dataset fixture, swapped for the test
// server's.
const realFileURL = "https://object-api.infra.data.gouv.fr/amf/LISTES_NOIRES/export_od_listesnoires_20261029080000_20261029083000.csv"

// upstream serves the dataset metadata at /dataset/ and the file at
// /file.csv; metadata builds the metadata from the server URL.
func upstream(t *testing.T, metadata func(srvURL string) string, file http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/dataset/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(metadata(srv.URL)))
	})
	mux.HandleFunc("/file.csv", file)
	return srv
}

func datasetPointingTo(t *testing.T, fileURL func(srvURL string) string) func(string) string {
	t.Helper()
	dataset := string(readFixture(t, "dataset.json"))
	return func(srvURL string) string {
		return strings.Replace(dataset, realFileURL, fileURL(srvURL), 1)
	}
}

func serveFixture(t *testing.T) http.HandlerFunc {
	t.Helper()
	csv := readFixture(t, csvFixture)
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(csv) }
}

func newFetcher(t *testing.T, srv *httptest.Server, cfg httpclient.Config) *Fetcher {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	cfg.RootCAs = pool
	cfg.AllowedHosts = []string{u.Host}
	cfg.Attempts = 1
	c, err := httpclient.New(cfg)
	if err != nil {
		t.Fatalf("httpclient.New() error = %v", err)
	}
	return NewFetcher(c, srv.URL+"/dataset/")
}

func TestFetch(t *testing.T) {
	t.Parallel()
	meta := datasetPointingTo(t, func(srvURL string) string { return srvURL + "/file.csv" })
	srv := upstream(t, meta, serveFixture(t))
	snap, err := newFetcher(t, srv, httpclient.Config{}).Fetch(t.Context())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snap.Source != Source || len(snap.Rows) != 23 {
		t.Errorf("Fetch() = %s with %d rows, want %s with 23", snap.Source, len(snap.Rows), Source)
	}
}

func TestFetchErrors(t *testing.T) {
	t.Parallel()
	toFile := func(srvURL string) string { return srvURL + "/file.csv" }
	tests := []struct {
		name string
		meta func(t *testing.T) func(string) string
		file func(t *testing.T) http.HandlerFunc
		cfg  httpclient.Config
		want error
	}{
		{
			name: "no CSV resource",
			meta: func(t *testing.T) func(string) string {
				return func(s string) string {
					return strings.Replace(datasetPointingTo(t, toFile)(s), `"format": "csv"`, `"format": "json"`, 1)
				}
			},
			want: ErrResource,
		},
		{
			name: "metadata not JSON",
			meta: func(*testing.T) func(string) string { return func(string) string { return "<html>" } },
			want: ErrResource,
		},
		{
			name: "file on a host not allowed",
			meta: func(t *testing.T) func(string) string {
				return datasetPointingTo(t, func(string) string { return realFileURL })
			},
			want: httpclient.ErrURLNotAllowed,
		},
		{
			name: "file of another format",
			file: func(*testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html></html>\n")) }
			},
			want: ErrFormat,
		},
		{
			name: "file too large",
			cfg:  httpclient.Config{MaxBodyBytes: 1 << 10},
			want: httpclient.ErrBodyTooLarge,
		},
		{
			name: "file too slow",
			cfg:  httpclient.Config{Timeout: 100 * time.Millisecond},
			file: func(*testing.T) http.HandlerFunc {
				return func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }
			},
			want: context.DeadlineExceeded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta, file := datasetPointingTo(t, toFile), serveFixture(t)
			if tt.meta != nil {
				meta = tt.meta(t)
			}
			if tt.file != nil {
				file = tt.file(t)
			}
			srv := upstream(t, meta, file)
			if _, err := newFetcher(t, srv, tt.cfg).Fetch(t.Context()); !errors.Is(err, tt.want) {
				t.Errorf("Fetch() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSpec(t *testing.T) {
	t.Parallel()
	spec := NewFetcher(nil, DatasetURL).Spec()
	if spec.Source != Source || len(spec.Columns) == 0 || spec.MinRows <= 0 {
		t.Errorf("Spec() = %+v", spec)
	}
}
