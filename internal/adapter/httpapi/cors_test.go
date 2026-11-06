// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
)

func TestDemoAllowsTheSiteOriginOnly(t *testing.T) {
	t.Parallel()
	allowsTheSiteOnly(t, func(t *testing.T, origin string) *httptest.ResponseRecorder {
		opts, _, _ := demoOptions(t)
		req := demoCall{body: validBody}.request(t)
		req.Header.Del("Origin")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return do(httpapi.NewHandler(opts), req)
	})
}

func TestSourcesAllowTheSiteOriginOnly(t *testing.T) {
	t.Parallel()
	allowsTheSiteOnly(t, func(t *testing.T, origin string) *httptest.ResponseRecorder {
		opts, _, _ := demoOptions(t)
		header := http.Header{}
		if origin != "" {
			header.Set("Origin", origin)
		}
		return serve(t, httpapi.NewHandler(opts), http.MethodGet, "/v1/sources", header)
	})
}

// allowsTheSiteOnly fails t unless the answer send gets from each origin
// lets a browser read it from the site's origin only, without credentials,
// and says that it depends on the origin.
func allowsTheSiteOnly(t *testing.T, send func(t *testing.T, origin string) *httptest.ResponseRecorder) {
	t.Helper()
	for _, tt := range []struct {
		name, origin, want string
	}{
		{"site", siteOrigin, siteOrigin},
		{"other site", "https://evil.example", ""},
		{"site over http", "http://arnakoff.com", ""},
		{"no origin", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := send(t, tt.origin)
			if got := res.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
			}
			if got := res.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want Origin: the answer depends on it", got)
			}
			if res.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Error("Access-Control-Allow-Credentials is set, want no credentials")
			}
		})
	}
}

func TestDemoExposesTheQuotaAndRequestFields(t *testing.T) {
	t.Parallel()
	opts, _, _ := demoOptions(t)

	res := do(httpapi.NewHandler(opts), demoCall{body: `{}`}.request(t))
	exposed := res.Header().Get("Access-Control-Expose-Headers")
	for _, f := range []string{"RateLimit", "RateLimit-Policy", "Retry-After", "X-Request-ID"} {
		if !strings.Contains(exposed, f) {
			t.Errorf("Access-Control-Expose-Headers = %q, want %s", exposed, f)
		}
	}
}

func TestDemoAnswersThePreflightOfTheSite(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, origin, method string
		allowed              bool
	}{
		{"site posting", siteOrigin, http.MethodPost, true},
		{"site deleting", siteOrigin, http.MethodDelete, false},
		{"other site", "https://evil.example", http.MethodPost, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, _, _ := demoOptions(t)
			res := serve(t, httpapi.NewHandler(opts), http.MethodOptions, "/v1/demo/check", http.Header{
				"Origin": {tt.origin}, "Access-Control-Request-Method": {tt.method},
				"Access-Control-Request-Headers": {"content-type"},
			})
			if res.Code != http.StatusNoContent {
				t.Errorf("status = %d, want 204", res.Code)
			}
			if tt.allowed {
				allowsThePost(t, res.Header())
			} else if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
			}
		})
	}
}

// allowsThePost fails t unless h allows a POST of JSON from the site, and
// lets the browser keep that answer.
func allowsThePost(t *testing.T, h http.Header) {
	t.Helper()
	if got := h.Get("Access-Control-Allow-Origin"); got != siteOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, siteOrigin)
	}
	if h.Get("Access-Control-Allow-Methods") != "POST" || h.Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("allows %q with %q, want POST with Content-Type",
			h.Get("Access-Control-Allow-Methods"), h.Get("Access-Control-Allow-Headers"))
	}
	if h.Get("Access-Control-Max-Age") == "" {
		t.Error("Access-Control-Max-Age is not set, want the preflight cached")
	}
}

func TestCORSIsOnlyOnTheSiteRoutes(t *testing.T) {
	t.Parallel()
	opts, _, _ := demoOptions(t)
	h := httpapi.NewHandler(opts)

	res := serve(t, h, http.MethodGet, "/v1/signals", http.Header{"Origin": {siteOrigin}})
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("GET /v1/signals: Access-Control-Allow-Origin = %q, want none", got)
	}
	res = serve(t, h, http.MethodOptions, "/v1/check", http.Header{
		"Origin": {siteOrigin}, "Access-Control-Request-Method": {http.MethodGet},
	})
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("preflight of /v1/check: Access-Control-Allow-Origin = %q, want none", got)
	}
}
