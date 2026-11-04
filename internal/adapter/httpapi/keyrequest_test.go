// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// loginToken is the token of a login link the fake requester accepts.
var loginToken = strings.Repeat("T", domain.LoginTokenLength)

// requester is a fake httpapi.KeyRequester: it fails with err, records
// the addresses it was asked to mail, and redeems loginToken only.
type requester struct {
	err  error
	mu   sync.Mutex
	sent []domain.Email
	key  domain.APIKey
}

func (q *requester) RequestLink(_ context.Context, to domain.Email) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.err != nil {
		return q.err
	}
	q.sent = append(q.sent, to)
	return nil
}

func (q *requester) Redeem(_ context.Context, tok string) (domain.APIKey, domain.KeyRecord, error) {
	if q.err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, q.err
	}
	if tok != loginToken {
		return domain.APIKey{}, domain.KeyRecord{}, app.ErrLoginLinkInvalid
	}
	key, err := domain.NewAPIKey(domain.KeyLive, bytes.NewReader(bytes.Repeat([]byte{0x2a, 0x11}, 40)))
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, err
	}
	q.key = key
	return key, domain.KeyRecord{ID: 1, Hint: key.Hint(), CreatedAt: time.Date(2026, 11, 3, 15, 5, 0, 0, paris)}, nil
}

func (q *requester) addresses() []domain.Email {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]domain.Email(nil), q.sent...)
}

// keyOptions returns the options of a handler whose key requests accept
// validToken from siteOrigin, with the fakes they use.
func keyOptions(t *testing.T) (httpapi.Options, *requester, *verifier) {
	t.Helper()
	opts, _ := options(t, true)
	q, v := &requester{}, &verifier{}
	opts.Keys = httpapi.KeyRequestOptions{
		Requests: q, Origin: siteOrigin, Verifier: v,
		Limit: frozenLimiter(1 << 30), AddressLimit: frozenLimiter(1 << 30),
	}
	return opts, q, v
}

// postCall is a POST of JSON from the site.
type postCall struct {
	path, body  string
	contentType string // "application/json" when empty
	refused     bool   // the contract refuses the request
}

// send posts call to a handler of opts and holds the exchange to the
// contract.
func (call postCall) send(t *testing.T, opts httpapi.Options) *httptest.ResponseRecorder {
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
		t.Errorf("POST %s %.60s breaks the contract: %v", call.path, call.body, err)
	}
	return rec
}

func (call postCall) request(t *testing.T) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, call.path, strings.NewReader(call.body))
	req.Header.Set("Origin", siteOrigin)
	ct := call.contentType
	if ct == "" {
		ct = "application/json"
	}
	req.Header.Set("Content-Type", ct)
	return req
}

// linkRequest is a request of a link for addr with validToken.
func linkRequest(addr string) postCall {
	return postCall{path: "/v1/keys/request", body: `{"email":"` + addr + `","turnstile_token":"` + validToken + `"}`}
}

func TestKeyRequestMailsALinkToAValidAddress(t *testing.T) {
	t.Parallel()
	opts, q, v := keyOptions(t)

	res := linkRequest("Dev@Exemple.fr").send(t, opts)
	if res.Code != http.StatusAccepted || res.Body.Len() != 0 {
		t.Fatalf("status = %d, body %q; want 202 with no body", res.Code, res.Body)
	}
	if got := q.addresses(); len(got) != 1 || got[0].String() != "Dev@exemple.fr" {
		t.Errorf("mailed %v, want the address of the body", got)
	}
	if got := v.tokens(); len(got) != 1 || got[0] != validToken {
		t.Errorf("verified %q, want the token of the body", got)
	}
	h := res.Header()
	if !strings.HasPrefix(h.Get("Ratelimit-Policy"), `"key_request";`) || h.Get("Access-Control-Allow-Origin") != siteOrigin {
		t.Errorf("headers = %v, want the key_request policy and CORS for the site", h)
	}
}

func TestKeyRequestAnswersTheSameOverTheAddressLimit(t *testing.T) {
	t.Parallel()
	opts, q, _ := keyOptions(t)
	opts.Keys.AddressLimit = frozenLimiter(1)

	for i, addr := range []string{"dev@exemple.fr", "DEV@exemple.fr", "autre@exemple.fr"} {
		if res := linkRequest(addr).send(t, opts); res.Code != http.StatusAccepted {
			t.Fatalf("request %d: status = %d, want 202", i, res.Code)
		}
	}
	got := q.addresses()
	if len(got) != 2 || got[0].String() != "dev@exemple.fr" || got[1].String() != "autre@exemple.fr" {
		t.Errorf("mailed %v, want one link per address, case ignored", got)
	}
}

func TestKeyRequestRefuses(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		call   postCall
		setup  func(*httpapi.Options, *requester, *verifier)
		status int
		code   string
	}{
		{"not json", postCall{path: "/v1/keys/request", body: "email=x", contentType: "text/plain", refused: true}, nil, 415, "unsupported_media_type"},
		{"no email", postCall{path: "/v1/keys/request", body: `{"turnstile_token":"` + validToken + `"}`, refused: true}, nil, 400, "invalid_body"},
		{"no token", postCall{path: "/v1/keys/request", body: `{"email":"dev@exemple.fr"}`, refused: true}, nil, 400, "invalid_body"},
		{"bad email", linkRequest("pas une adresse"), nil, 422, "invalid_input"},
		{"too large", postCall{path: "/v1/keys/request", body: `{"email":"` + strings.Repeat("a", 9<<10) + `"}`, refused: true}, nil, 413, "content_too_large"},
		{"bad challenge", postCall{path: "/v1/keys/request", body: `{"email":"dev@exemple.fr","turnstile_token":"0.forged"}`}, nil, 403, "challenge_failed"},
		{"challenge down", linkRequest("dev@exemple.fr"), func(_ *httpapi.Options, _ *requester, v *verifier) { v.err = errors.New("down") }, 503, "challenge_unavailable"},
		{"mail down", linkRequest("dev@exemple.fr"), func(_ *httpapi.Options, q *requester, _ *verifier) { q.err = errors.New("smtp down") }, 503, "key_request_unavailable"},
		{"off", linkRequest("dev@exemple.fr"), func(o *httpapi.Options, _ *requester, _ *verifier) { o.Keys.Requests = nil }, 503, "key_request_unavailable"},
		{"no verifier", linkRequest("dev@exemple.fr"), func(o *httpapi.Options, _ *requester, _ *verifier) { o.Keys.Verifier = nil }, 503, "key_request_unavailable"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, q, v := keyOptions(t)
			if tt.setup != nil {
				tt.setup(&opts, q, v)
			}
			res := tt.call.send(t, opts)
			if res.Code != tt.status || decode(t, res)["code"] != tt.code {
				t.Fatalf("status = %d, body %s; want %d %s", res.Code, res.Body, tt.status, tt.code)
			}
			if len(q.addresses()) != 0 {
				t.Error("a link was mailed")
			}
			if strings.Contains(res.Body.String(), "exemple.fr") {
				t.Errorf("body %s quotes the address", res.Body)
			}
		})
	}
}

func TestKeyRoutesShareTheQuotaOfTheClient(t *testing.T) {
	t.Parallel()
	opts, _, _ := keyOptions(t)
	opts.Keys.Limit = frozenLimiter(1)
	h := httpapi.NewHandler(opts)

	if res := do(h, linkRequest("dev@exemple.fr").request(t)); res.Code != http.StatusAccepted {
		t.Fatalf("first request: status = %d, want 202", res.Code)
	}
	res := do(h, redeem(loginToken).request(t))
	if res.Code != http.StatusTooManyRequests || res.Header().Get("Retry-After") == "" ||
		!strings.HasPrefix(res.Header().Get("Ratelimit-Policy"), `"key_request";`) {
		t.Fatalf("second request: status = %d, headers %v; want 429 under key_request", res.Code, res.Header())
	}
}

// redeem is a redemption of tok.
func redeem(tok string) postCall {
	return postCall{path: "/v1/keys/redeem", body: `{"token":"` + tok + `"}`}
}

func TestKeyRedeemAnswersTheKeyOnce(t *testing.T) {
	t.Parallel()
	opts, q, _ := keyOptions(t)

	res := redeem(loginToken).send(t, opts)
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s; want 201", res.Code, res.Body)
	}
	body := decode(t, res)
	if body["api_key"] != q.key.String() || body["hint"] != q.key.Hint() || body["created_at"] != "2026-11-03T14:05:00Z" {
		t.Errorf("body = %v, want the key, its hint and its creation time in UTC", body)
	}
	h := res.Header()
	if h.Get("Cache-Control") != "no-store" || h.Get("Access-Control-Allow-Origin") != siteOrigin {
		t.Errorf("headers = %v, want no-store and CORS for the site", h)
	}
}

func TestKeyRedeemRefuses(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		call   postCall
		err    error
		off    bool
		status int
		code   string
	}{
		{"bad link", redeem(strings.Repeat("X", domain.LoginTokenLength)), nil, false, 422, "invalid_login_link"},
		{"no token", postCall{path: "/v1/keys/redeem", body: `{}`, refused: true}, nil, false, 400, "invalid_body"},
		{"not an object", postCall{path: "/v1/keys/redeem", body: `["x"]`, refused: true}, nil, false, 400, "invalid_body"},
		{"store down", redeem(loginToken), errors.New("database down"), false, 503, "key_request_unavailable"},
		{"off", redeem(loginToken), nil, true, 503, "key_request_unavailable"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, q, _ := keyOptions(t)
			q.err = tt.err
			if tt.off {
				opts.Keys.Requests = nil
			}
			res := tt.call.send(t, opts)
			if res.Code != tt.status || decode(t, res)["code"] != tt.code {
				t.Fatalf("status = %d, body %s; want %d %s", res.Code, res.Body, tt.status, tt.code)
			}
		})
	}
}

func TestKeyRoutesAnswerTheSitePreflight(t *testing.T) {
	t.Parallel()
	opts, _, _ := keyOptions(t)
	h := httpapi.NewHandler(opts)
	for _, path := range []string{"/v1/keys/request", "/v1/keys/redeem"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, path, nil)
		req.Header.Set("Origin", siteOrigin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		res := do(h, req)
		if res.Code != http.StatusNoContent || res.Header().Get("Access-Control-Allow-Origin") != siteOrigin {
			t.Errorf("OPTIONS %s = %d %v, want 204 allowing the site", path, res.Code, res.Header())
		}
	}
}
