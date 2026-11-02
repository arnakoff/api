// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/app"
)

// limitedOptions returns options limiting each key and each client
// address to one request a minute.
func limitedOptions(t *testing.T) httpapi.Options {
	t.Helper()
	opts, _ := options(t, true)
	opts.KeyLimit = frozenLimiter(1)
	opts.IPLimit = frozenLimiter(1)
	return opts
}

// fromAddr returns a request to target from the client at remote; a POST
// carries a batch of one item.
func fromAddr(t *testing.T, method, target, remote string) *http.Request {
	t.Helper()
	body := ""
	if method == http.MethodPost {
		body = `{"items":[{"q":"armen.ltd"}]}`
	}
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.RemoteAddr = remote
	return req
}

// limitHeaders checks the RateLimit fields of res and its Retry-After.
func limitHeaders(t *testing.T, res *httptest.ResponseRecorder, policy, limit, retryAfter string) {
	t.Helper()
	for name, want := range map[string]string{
		"RateLimit-Policy": policy, "RateLimit": limit, "Retry-After": retryAfter,
	} {
		if got := res.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestCheckRoutesAreLimitedPerKey(t *testing.T) {
	t.Parallel()
	for _, route := range []string{"GET /v1/check?q=armen.ltd", "POST /v1/check/batch"} {
		t.Run(route, func(t *testing.T) {
			t.Parallel()
			h := httpapi.NewHandler(limitedOptions(t))
			method, target, _ := strings.Cut(route, " ")
			first := do(h, authorize(fromAddr(t, method, target, "203.0.113.7:1")))
			if first.Code != http.StatusOK {
				t.Fatalf("first request: status %d, want 200", first.Code)
			}
			limitHeaders(t, first, `"key";q=1;w=60`, `"key";r=0;t=60`, "")
			// Another address: the key, not the address, is limited.
			second := do(h, authorize(fromAddr(t, method, target, "198.51.100.1:1")))
			if second.Code != http.StatusTooManyRequests {
				t.Fatalf("second request: status %d, want 429", second.Code)
			}
			limitHeaders(t, second, `"key";q=1;w=60`, `"key";r=0;t=60`, "60")
			if code := decode(t, second)["code"]; code != "too_many_requests" {
				t.Errorf("code = %v, want too_many_requests", code)
			}
		})
	}
}

func TestARefusedKeySpendsNoQuota(t *testing.T) {
	t.Parallel()
	opts := limitedOptions(t)
	opts.Auth = keyring{err: app.ErrKeyNotFound}
	h := httpapi.NewHandler(opts)
	for range 2 {
		res := do(h, authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1")))
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", res.Code)
		}
		if got := res.Header().Get("RateLimit"); got != "" {
			t.Errorf("RateLimit = %q on a refused key, want none", got)
		}
	}
}

func TestReferenceRoutesAreLimitedPerAddress(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"/v1/sources", "/v1/signals"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			opts := limitedOptions(t)
			opts.Sources = sourceLister{}
			h := httpapi.NewHandler(opts)
			first := do(h, fromAddr(t, http.MethodGet, target, "203.0.113.7:1"))
			if first.Code != http.StatusOK {
				t.Fatalf("first request: status %d, want 200", first.Code)
			}
			limitHeaders(t, first, `"ip";q=1;w=60`, `"ip";r=0;t=60`, "")
			if res := do(h, fromAddr(t, http.MethodGet, target, "203.0.113.7:2")); res.Code != http.StatusTooManyRequests {
				t.Errorf("same address: status %d, want 429", res.Code)
			}
			if res := do(h, fromAddr(t, http.MethodGet, target, "198.51.100.1:1")); res.Code != http.StatusOK {
				t.Errorf("another address: status %d, want 200", res.Code)
			}
		})
	}
}

func TestTheAddressBehindATrustedProxyIsLimited(t *testing.T) {
	t.Parallel()
	opts := limitedOptions(t)
	opts.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	h := httpapi.NewHandler(opts)
	for _, client := range []string{"203.0.113.7", "198.51.100.1"} {
		req := fromAddr(t, http.MethodGet, "/v1/signals", "10.0.0.1:443")
		req.Header.Set("X-Forwarded-For", client)
		if res := do(h, req); res.Code != http.StatusOK {
			t.Errorf("client %s behind the proxy: status %d, want 200", client, res.Code)
		}
	}
}

func TestProbesAreNeverLimited(t *testing.T) {
	t.Parallel()
	h := httpapi.NewHandler(limitedOptions(t))
	for _, target := range []string{"/healthz", "/readyz", "/healthz"} {
		res := do(h, fromAddr(t, http.MethodGet, target, "203.0.113.7:1"))
		if res.Code != http.StatusOK || res.Header().Get("RateLimit") != "" {
			t.Errorf("%s: status %d, RateLimit %q, want 200 and none", target, res.Code, res.Header().Get("RateLimit"))
		}
	}
}

func TestUnsetLimitersApplyTheDefaults(t *testing.T) {
	t.Parallel()
	opts, _ := options(t, true)
	opts.KeyLimit, opts.IPLimit = nil, nil
	h := httpapi.NewHandler(opts)
	res := do(h, authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1")))
	if got, want := res.Header().Get("RateLimit-Policy"), `"key";q=60;w=60`; got != want {
		t.Errorf("check: RateLimit-Policy = %q, want %q", got, want)
	}
	res = do(h, fromAddr(t, http.MethodGet, "/v1/signals", "203.0.113.7:1"))
	if got, want := res.Header().Get("RateLimit-Policy"), `"ip";q=30;w=60`; got != want {
		t.Errorf("signals: RateLimit-Policy = %q, want %q", got, want)
	}
}

func TestALimitedAnswerHoldsToTheContract(t *testing.T) {
	t.Parallel()
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	h := httpapi.NewHandler(limitedOptions(t))
	for _, status := range []int{http.StatusOK, http.StatusTooManyRequests} {
		req := authorize(fromAddr(t, http.MethodGet, "/v1/check?q=armen.ltd", "203.0.113.7:1"))
		res := do(h, req)
		if res.Code != status {
			t.Fatalf("status %d, want %d", res.Code, status)
		}
		if err := ct.Check(t.Context(), req, res.Result()); err != nil {
			t.Errorf("the %d answer breaks the contract: %v", res.Code, err)
		}
	}
}
