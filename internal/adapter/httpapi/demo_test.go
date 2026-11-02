// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/domain"
)

// siteOrigin is the origin of the site in the demo tests.
const siteOrigin = "https://arnakoff.com"

// validToken is the token the fake verifier accepts.
const validToken = "0.valid-t0ken" //nolint:gosec // G101: a fake token of the tests.

// validBody is a demo check of a domain with validToken.
const validBody = `{"q":"armen.ltd","turnstile_token":"` + validToken + `"}`

// verifier is a fake challenge verifier: it fails with err, or accepts
// validToken only, and records the tokens it was asked about.
type verifier struct {
	err   error
	mu    sync.Mutex
	asked []string
}

func (v *verifier) Verify(_ context.Context, token string) (bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.asked = append(v.asked, token)
	return token == validToken && v.err == nil, v.err
}

func (v *verifier) tokens() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]string(nil), v.asked...)
}

// demoCall is a POST /v1/demo/check to send.
type demoCall struct {
	body        string
	contentType string // "application/json" when empty
	refused     bool   // the contract refuses the request
}

// demoOptions returns the options of a handler whose demo accepts
// validToken from siteOrigin, with the fakes it uses.
func demoOptions(t *testing.T) (httpapi.Options, *checker, *verifier) {
	t.Helper()
	opts, _ := options(t, true)
	c, v := &checker{}, &verifier{}
	opts.Check = c
	opts.Demo = httpapi.DemoOptions{
		Origin: siteOrigin, Verifier: v, Limit: frozenLimiter(1 << 30), GlobalLimit: frozenLimiter(1 << 30),
	}
	return opts, c, v
}

// send posts call to a handler of opts and holds the exchange to the
// contract.
func (call demoCall) send(t *testing.T, opts httpapi.Options) *httptest.ResponseRecorder {
	t.Helper()
	rec := do(httpapi.NewHandler(opts), call.request(t))
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	hold := ct.Check
	if call.refused {
		hold = ct.CheckRefused
	}
	if err := hold(t.Context(), call.request(t), rec.Result()); err != nil {
		t.Errorf("POST %.60s breaks the contract: %v", call.body, err)
	}
	return rec
}

func (call demoCall) request(t *testing.T) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/demo/check", strings.NewReader(call.body))
	req.Header.Set("Origin", siteOrigin)
	ct := call.contentType
	if ct == "" {
		ct = "application/json"
	}
	req.Header.Set("Content-Type", ct)
	return req
}

func TestDemoAnswersTheReportOfAValidToken(t *testing.T) {
	t.Parallel()
	opts, c, v := demoOptions(t)

	res := demoCall{body: validBody}.send(t, opts)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", res.Code, res.Body)
	}
	if in := c.inputs(); len(in) != 1 || in[0].Kind() != domain.KindDomain {
		t.Errorf("checked %v, want the domain armen.ltd", in)
	}
	if got := v.tokens(); len(got) != 1 || got[0] != validToken {
		t.Errorf("verified %q, want the token of the body", got)
	}
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store: the input may be personal data", got)
	}
	if got := res.Header().Get("Ratelimit-Policy"); !strings.HasPrefix(got, `"demo";`) {
		t.Errorf("RateLimit-Policy = %q, want the demo policy", got)
	}
}

func TestDemoRefusesAToken(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body string
		err        error
		noVerifier bool
		status     int
		code       string
		refused    bool
	}{
		{"refused", `{"q":"armen.ltd","turnstile_token":"0.replayed"}`, nil, false, 403, "challenge_failed", false},
		{"missing", `{"q":"armen.ltd"}`, nil, false, 403, "challenge_failed", true},
		{"unchecked", validBody, errors.New("down"), false, 503, "challenge_unavailable", false},
		{"no verifier", validBody, nil, true, 503, "challenge_unavailable", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, c, v := demoOptions(t)
			v.err = tt.err
			if tt.noVerifier {
				opts.Demo.Verifier = nil
			}

			res := demoCall{body: tt.body, refused: tt.refused}.send(t, opts)
			if res.Code != tt.status || decode(t, res)["code"] != tt.code {
				t.Errorf("status = %d, want %d %s", res.Code, tt.status, tt.code)
			}
			if len(c.inputs()) != 0 {
				t.Error("the input was checked, want no check without a valid token")
			}
		})
	}
}

func TestDemoRefusesAnInvalidBodyBeforeTheToken(t *testing.T) {
	t.Parallel()
	token := `,"turnstile_token":"` + validToken + `"}`
	for _, tt := range []struct {
		name        string
		call        demoCall
		status      int
		code        string
		wantPayload string
	}{
		{"not JSON", demoCall{body: `q=armen.ltd`, contentType: "text/plain", refused: true}, 415, "unsupported_media_type", ""},
		{"malformed", demoCall{body: `{"q":`, refused: true}, 400, "invalid_body", "turnstile_token"},
		{"two values", demoCall{body: `{"q":"a.fr"}{}`, refused: true}, 400, "invalid_body", "turnstile_token"},
		{"no q", demoCall{body: `{"q":""` + token, refused: true}, 400, "missing_parameter", ""},
		{"unknown type", demoCall{body: `{"q":"a.fr","type":"auto"` + token, refused: true}, 400, "invalid_parameter", ""},
		{"invalid input", demoCall{body: `{"q":"FR76 0000","type":"iban"` + token}, 422, "invalid_input", ""},
		{"too large", demoCall{body: `{"q":"` + strings.Repeat("a", 9<<10) + `"` + token}, 413, "content_too_large", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, _, v := demoOptions(t)

			res := tt.call.send(t, opts)
			body := decode(t, res)
			if res.Code != tt.status || body["code"] != tt.code {
				t.Errorf("status = %d %v, want %d %s", res.Code, body["code"], tt.status, tt.code)
			}
			if detail, _ := body["detail"].(string); !strings.Contains(detail, tt.wantPayload) {
				t.Errorf("detail = %q, want it to name %s", detail, tt.wantPayload)
			}
			if got := v.tokens(); len(got) != 0 {
				t.Errorf("verified %q, want no token spent on an invalid body", got)
			}
		})
	}
}

func TestDemoLimitsEachAddress(t *testing.T) {
	t.Parallel()
	opts, _, _ := demoOptions(t)
	opts.Demo.Limit = frozenLimiter(1)
	call := demoCall{body: validBody}

	if res := call.send(t, opts); res.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", res.Code)
	}
	res := call.send(t, opts)
	if res.Code != http.StatusTooManyRequests || !strings.HasPrefix(res.Header().Get("Ratelimit-Policy"), `"demo";`) {
		t.Errorf("second status = %d, policy %q, want 429 of the demo policy", res.Code, res.Header().Get("Ratelimit-Policy"))
	}
}

func TestDemoLimitsEveryClientTogether(t *testing.T) {
	t.Parallel()
	opts, _, _ := demoOptions(t)
	opts.Demo.GlobalLimit = frozenLimiter(1)
	h := httpapi.NewHandler(opts)
	body := validBody
	from := func(addr string) *httptest.ResponseRecorder {
		req := demoCall{body: body}.request(t)
		req.RemoteAddr = addr
		return do(h, req)
	}

	if res := from("192.0.2.1:1234"); res.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", res.Code)
	}
	res := from("198.51.100.2:1234")
	if res.Code != http.StatusTooManyRequests || !strings.HasPrefix(res.Header().Get("Ratelimit-Policy"), `"demo_global";`) {
		t.Errorf("status from another address = %d, policy %q, want 429 of the global policy",
			res.Code, res.Header().Get("Ratelimit-Policy"))
	}
}

func TestDemoSpendsTheGlobalQuotaOnVerifiedTokensOnly(t *testing.T) {
	t.Parallel()
	opts, _, _ := demoOptions(t)
	opts.Demo.GlobalLimit = frozenLimiter(1)
	h := httpapi.NewHandler(opts)
	from := func(addr, token string) *httptest.ResponseRecorder {
		req := demoCall{body: `{"q":"armen.ltd","turnstile_token":"` + token + `"}`}.request(t)
		req.RemoteAddr = addr
		return do(h, req)
	}

	if res := from("192.0.2.1:1234", "0.forged"); res.Code != http.StatusForbidden {
		t.Fatalf("forged token status = %d, want 403", res.Code)
	}
	if res := from("198.51.100.2:1234", validToken); res.Code != http.StatusOK {
		t.Errorf("valid token status = %d, want 200: a forged token must not spend the global quota", res.Code)
	}
}
