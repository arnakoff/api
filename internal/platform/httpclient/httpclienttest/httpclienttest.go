// SPDX-License-Identifier: AGPL-3.0-only

// Package httpclienttest builds httpclient.Client values for tests against
// an httptest TLS server.
package httpclienttest

import (
	"crypto/x509"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/arnakoff/api/internal/platform/httpclient"
)

// NewClient returns a client built from cfg that trusts and allows the TLS
// server srv only, with one attempt per call.
func NewClient(tb testing.TB, srv *httptest.Server, cfg httpclient.Config) *httpclient.Client {
	tb.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		tb.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	cfg.RootCAs = pool
	cfg.AllowedHosts = []string{u.Host}
	cfg.Attempts = 1
	c, err := httpclient.New(cfg)
	if err != nil {
		tb.Fatalf("httpclient.New() error = %v", err)
	}
	return c
}
