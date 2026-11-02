// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
)

func TestEveryResponseCarriesTheSecurityHeaders(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	h := httpapi.NewHandler(opts)
	for _, target := range []string{"/healthz", "/readyz", "/nowhere"} {
		res := serve(t, h, http.MethodGet, target, nil)
		if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, want nosniff", target, got)
		}
		if got := res.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", target, got)
		}
	}
}

func TestUnmatchedRequestsGetAProblemNeverHTML(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	h := httpapi.NewHandler(opts)
	for _, tt := range []struct {
		method, target string
		status         int
		code           string
		allow          string
	}{
		{http.MethodGet, "/nowhere", http.StatusNotFound, "not_found", ""},
		{http.MethodGet, "/healthz/", http.StatusNotFound, "not_found", ""},
		{http.MethodGet, "//healthz", http.StatusNotFound, "not_found", ""},
		{http.MethodGet, "/x/../healthz", http.StatusNotFound, "not_found", ""},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed, "method_not_allowed", "GET, HEAD"},
	} {
		res := serve(t, h, tt.method, tt.target, nil)
		hasNoHTML(t, res)
		if res.Code != tt.status {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.target, res.Code, tt.status)
		}
		if ct := res.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Errorf("%s %s: Content-Type = %q, want application/problem+json", tt.method, tt.target, ct)
		}
		if got := res.Header().Get("Allow"); got != tt.allow {
			t.Errorf("%s %s: Allow = %q, want %q", tt.method, tt.target, got, tt.allow)
		}
		p := decode(t, res)
		if p["code"] != tt.code || p["type"] != "/problemes/"+tt.code || p["status"] != float64(tt.status) {
			t.Errorf("%s %s: problem = %v, want code %s", tt.method, tt.target, p, tt.code)
		}
	}
}
