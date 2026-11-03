// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// authCase is a request to a route that requires a key, and the answer
// expected.
type authCase struct {
	name      string
	header    []string // the Authorization field lines
	auth      httpapi.Authenticator
	unsetAuth bool // Options.Auth left nil
	status    int
	code      string
	challenge string // WWW-Authenticate, none when empty
}

func TestCheckRoutesRequireAKey(t *testing.T) {
	t.Parallel()

	const (
		challenge = `Bearer realm="arnakoff"`
		invalid   = `Bearer realm="arnakoff", error="invalid_token"`
	)
	tests := []authCase{
		{name: "valid key", header: []string{"Bearer " + testKey}, status: http.StatusOK},
		{name: "scheme in any case", header: []string{"bearer " + testKey}, status: http.StatusOK},
		{name: "no header", status: http.StatusUnauthorized, code: "missing_api_key", challenge: challenge},
		{name: "other scheme", header: []string{"Basic YTpi"}, status: http.StatusUnauthorized, code: "missing_api_key", challenge: challenge},
		{name: "empty token", header: []string{"Bearer "}, status: http.StatusUnauthorized, code: "invalid_api_key", challenge: invalid},
		{name: "two fields", header: []string{"Bearer " + testKey, "Bearer " + testKey}, status: http.StatusUnauthorized, code: "invalid_api_key", challenge: invalid},
		{name: "malformed key", header: []string{"Bearer nope"}, auth: keyring{err: domain.ErrInvalidAPIKey}, status: http.StatusUnauthorized, code: "invalid_api_key", challenge: invalid},
		{name: "unknown key", header: []string{"Bearer " + testKey}, auth: keyring{err: app.ErrKeyNotFound}, status: http.StatusUnauthorized, code: "invalid_api_key", challenge: invalid},
		{name: "revoked key", header: []string{"Bearer " + testKey}, auth: keyring{err: app.ErrKeyRevoked}, status: http.StatusUnauthorized, code: "invalid_api_key", challenge: invalid},
		{name: "store down", header: []string{"Bearer " + testKey}, auth: keyring{err: errors.New("dial tcp: refused")}, status: http.StatusServiceUnavailable, code: "authentication_unavailable"},
		{name: "no authenticator", header: []string{"Bearer " + testKey}, unsetAuth: true, status: http.StatusServiceUnavailable, code: "authentication_unavailable"},
	}
	for _, tt := range tests {
		for _, route := range []string{"GET /v1/check?q=armen.ltd", "POST /v1/check/batch", "POST /mcp"} {
			t.Run(tt.name+" "+route, func(t *testing.T) {
				t.Parallel()
				tt.assert(t, route)
			})
		}
	}
}

func (tt authCase) assert(t *testing.T, route string) {
	t.Helper()
	opts, _ := options(t, true)
	switch {
	case tt.unsetAuth:
		opts.Auth = nil
	case tt.auth != nil:
		opts.Auth = tt.auth
	}
	method, target, _ := strings.Cut(route, " ")
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(`{"items":[{"q":"armen.ltd"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header["Authorization"] = tt.header
	res := httptest.NewRecorder()
	httpapi.NewHandler(opts).ServeHTTP(res, req)

	if res.Code != tt.status {
		t.Fatalf("status = %d, want %d", res.Code, tt.status)
	}
	if got := res.Header().Get("WWW-Authenticate"); got != tt.challenge {
		t.Errorf("WWW-Authenticate = %q, want %q", got, tt.challenge)
	}
	if tt.code == "" {
		return
	}
	if got := decode(t, res)["code"]; got != tt.code {
		t.Errorf("code = %v, want %s", got, tt.code)
	}
}

func TestReferenceRoutesNeedNoKey(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	opts.Auth = keyring{err: app.ErrKeyNotFound}
	h := httpapi.NewHandler(opts)
	for _, target := range []string{"/healthz", "/readyz", "/v1/signals"} {
		if res := serve(t, h, http.MethodGet, target, nil); res.Code != http.StatusOK {
			t.Errorf("GET %s without a key: status = %d, want 200", target, res.Code)
		}
	}
}

func TestAKeyRefusalHoldsToTheContract(t *testing.T) {
	t.Parallel()

	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		header  string
		auth    httpapi.Authenticator
		refused bool // the contract refuses the request
	}{
		{"missing", "", nil, true},
		{"invalid", "Bearer " + testKey, keyring{err: app.ErrKeyRevoked}, false},
		{"unavailable", "Bearer " + testKey, keyring{err: errors.New("down")}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, _ := options(t, true)
			if tt.auth != nil {
				opts.Auth = tt.auth
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/check?q=armen.ltd", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			res := httptest.NewRecorder()
			httpapi.NewHandler(opts).ServeHTTP(res, req)
			hold := ct.Check
			if tt.refused {
				hold = ct.CheckRefused
			}
			if err := hold(t.Context(), req, res.Result()); err != nil {
				t.Errorf("the %d answer breaks the contract: %v", res.Code, err)
			}
		})
	}
}

func TestAStoreFailureIsLoggedWithoutTheKey(t *testing.T) {
	t.Parallel()

	opts, logs := options(t, true)
	opts.Auth = keyring{err: errors.New("dial tcp 10.0.0.5:5432: connection refused")}
	res := serve(t, httpapi.NewHandler(opts), http.MethodGet, "/v1/check?q=armen.ltd",
		http.Header{"Authorization": {"Bearer " + testKey}})
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.Code)
	}
	if !strings.Contains(logs.String(), "connection refused") {
		t.Errorf("log %s does not name the store failure", logs.String())
	}
	if strings.Contains(logs.String(), testKey) {
		t.Errorf("log %s quotes the API key", logs.String())
	}
}

func TestMCPIsNotServedWhenUnset(t *testing.T) {
	t.Parallel()
	opts, _ := options(t, true)
	opts.MCP = nil
	res := do(httpapi.NewHandler(opts), authorize(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", nil)))
	if res.Code != http.StatusNotFound {
		t.Errorf("POST /mcp without an MCP endpoint: status %d, want 404", res.Code)
	}
}
