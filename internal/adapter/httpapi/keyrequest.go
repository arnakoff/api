// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/ratelimit"
)

// KeyRequester issues free API keys by login link.
type KeyRequester interface {
	// RequestLink mails a login link to to.
	RequestLink(ctx context.Context, to domain.Email) error
	// Redeem issues a key for the login link of token, or fails with
	// app.ErrLoginLinkInvalid.
	Redeem(ctx context.Context, token string) (domain.APIKey, domain.KeyRecord, error)
}

// KeyRequestOptions configures POST /v1/keys/request and
// POST /v1/keys/redeem, the free keys requested from the site.
type KeyRequestOptions struct {
	// Requests mails the links and redeems them; nil answers both routes
	// with a 503.
	Requests KeyRequester
	// Origin is the site's origin, the only one CORS allows; empty allows
	// none.
	Origin string
	// Verifier checks the anti-abuse token of each link request; nil
	// answers them with a 503.
	Verifier ChallengeVerifier
	// Limit limits the requests of each client address to both routes;
	// nil applies config.DefaultRateLimitKeyRequest a minute.
	Limit Limiter
	// AddressLimit limits the links mailed to each email address; nil
	// applies config.DefaultRateLimitKeyRequestAddress an hour. Over it, a
	// request is answered as any other, and no link is mailed.
	AddressLimit Limiter
	// GlobalLimit limits the links mailed to every address together; nil
	// applies config.DefaultRateLimitKeyRequestGlobal an hour. Over it, a
	// request is answered 503 and a warning is logged for the alert.
	GlobalLimit Limiter
}

// keyRequestPolicy is the name of the quota of the key request routes.
const keyRequestPolicy = "key_request"

// Bounds of the bodies of the key request routes: an address and a
// Turnstile token, or a login token.
const (
	keyRequestMaxBytes = 8 << 10
	keyRedeemMaxBytes  = 1 << 10
)

// keyRequestBody is a request as the contract's KeyRequest schema
// describes it; unknown fields are ignored.
type keyRequestBody struct {
	Email string `json:"email"`
	Token string `json:"turnstile_token"`
}

// keyRedeemBody is a request as the contract's KeyRedeemRequest schema
// describes it.
type keyRedeemBody struct {
	Token string `json:"token"`
}

// issuedKey is a key as the contract's IssuedKey schema describes it.
type issuedKey struct {
	APIKey    string    `json:"api_key"`
	Hint      string    `json:"hint"`
	CreatedAt time.Time `json:"created_at"`
}

// keyRequestGate admits to the key request routes the requests within the quota
// of their client address.
type keyRequestGate struct {
	opts     KeyRequestOptions
	logger   *slog.Logger
	ps       problems
	perAddr  *quota
	perEmail Limiter
	global   Limiter
	clientOf func(*http.Request) string
}

// newKeyRequestGate returns the gate opts.Keys configures, knowing each
// client by clientOf.
func newKeyRequestGate(opts Options, ps problems, clientOf func(*http.Request) string) *keyRequestGate {
	return &keyRequestGate{
		opts: opts.Keys, logger: opts.Logger, ps: ps, clientOf: clientOf,
		perEmail: hourlyOr(opts.Keys.AddressLimit, config.DefaultRateLimitKeyRequestAddress),
		global:   hourlyOr(opts.Keys.GlobalLimit, config.DefaultRateLimitKeyRequestGlobal),
		perAddr:  newQuota(limiterOr(opts.Keys.Limit, config.DefaultRateLimitKeyRequest), keyRequestPolicy, ps),
	}
}

// routes adds the key request routes, and their CORS preflights, to mux.
func (g *keyRequestGate) routes(mux *http.ServeMux) {
	origin := g.opts.Origin
	mux.Handle("POST /v1/keys/request", allowOrigin(origin, http.HandlerFunc(g.request)))
	mux.Handle("OPTIONS /v1/keys/request", preflight(origin))
	mux.Handle("POST /v1/keys/redeem", allowOrigin(origin, http.HandlerFunc(g.redeem)))
	mux.Handle("OPTIONS /v1/keys/redeem", preflight(origin))
}

// request answers POST /v1/keys/request: a 202 once the body's address is
// valid and its token proven, whether a link is mailed or the address
// already had its share. A body is refused before its token is spent.
func (g *keyRequestGate) request(w http.ResponseWriter, r *http.Request) {
	if g.opts.Requests == nil || g.opts.Verifier == nil {
		g.ps.write(w, r, keyRequestUnavailable)
		return
	}
	if !g.perAddr.admit(w, r, g.clientOf(r)) {
		return
	}
	to, token, p, ok := linkRequestOf(w, r)
	if !ok {
		g.ps.write(w, r, p)
		return
	}
	if p, ok := verifyChallenge(r.Context(), g.opts.Verifier, g.logger, token); !ok {
		g.ps.write(w, r, p)
		return
	}
	if p, ok := g.mailLink(r.Context(), to); !ok {
		g.ps.write(w, r, p)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// hourlyOr returns l, or a limiter of n an hour when l is nil.
func hourlyOr(l Limiter, n int) Limiter {
	if l == nil {
		return ratelimit.PerHour(n, time.Now)
	}
	return l
}

// mailLink mails a login link to to within the global cap, unless to
// already had its share, and returns the problem when it cannot. The cap
// is checked first, so that over it every address gets the same answer;
// a link held back by the address limit gives its share of the cap back.
func (g *keyRequestGate) mailLink(ctx context.Context, to domain.Email) (problem, bool) {
	if !g.global.Allow(everyClient).Allowed {
		g.logger.WarnContext(ctx, "login link cap reached", "quota", g.global.Policy().Quota)
		return keyRequestUnavailable, false
	}
	if !g.perEmail.Allow(accountKey(to)).Allowed {
		g.global.Refund(everyClient)
		return problem{}, true
	}
	if err := g.opts.Requests.RequestLink(ctx, to); err != nil {
		g.logger.ErrorContext(ctx, "login link not sent", "error", err.Error())
		return keyRequestUnavailable, false
	}
	return problem{}, true
}

// linkRequestOf returns the address and the token the body of r holds, or
// the problem refusing it.
func linkRequestOf(w http.ResponseWriter, r *http.Request) (domain.Email, string, problem, bool) {
	var body keyRequestBody
	if p, ok := decodeBody(w, r, keyRequestMaxBytes, &body, invalidKeyRequestBody); !ok {
		return domain.Email{}, "", p, false
	}
	if body.Email == "" || body.Token == "" {
		return domain.Email{}, "", invalidKeyRequestBody, false
	}
	to, err := domain.ParseEmail(body.Email)
	if err != nil {
		return domain.Email{}, "", invalidInputOf[domain.KindEmail], false
	}
	return to, body.Token, problem{}, true
}

// accountKey is the key of the address limit of to: the digest of its
// account, so that the limiter does not keep addresses.
func accountKey(to domain.Email) string {
	sum := sha256.Sum256([]byte(to.Account()))
	return hex.EncodeToString(sum[:])
}

// redeem answers POST /v1/keys/redeem: the key the body's login token
// gives, shown this once.
func (g *keyRequestGate) redeem(w http.ResponseWriter, r *http.Request) {
	if g.opts.Requests == nil {
		g.ps.write(w, r, keyRequestUnavailable)
		return
	}
	if !g.perAddr.admit(w, r, g.clientOf(r)) {
		return
	}
	var body keyRedeemBody
	if p, ok := decodeBody(w, r, keyRedeemMaxBytes, &body, invalidKeyRedeemBody); !ok {
		g.ps.write(w, r, p)
		return
	}
	if body.Token == "" {
		g.ps.write(w, r, invalidKeyRedeemBody)
		return
	}
	key, rec, err := g.opts.Requests.Redeem(r.Context(), body.Token)
	switch {
	case errors.Is(err, app.ErrLoginLinkInvalid):
		g.ps.write(w, r, invalidLoginLink)
	case err != nil:
		g.logger.ErrorContext(r.Context(), "API key not issued", "error", err.Error())
		g.ps.write(w, r, keyRequestUnavailable)
	default:
		writeJSON(w, "application/json", http.StatusCreated, issuedKey{
			APIKey: key.String(), Hint: rec.Hint, CreatedAt: rec.CreatedAt.UTC(),
		})
	}
}
