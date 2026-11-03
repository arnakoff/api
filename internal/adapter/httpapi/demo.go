// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

// ChallengeVerifier tells whether an anti-abuse token, solved by a
// visitor of the site, is valid.
type ChallengeVerifier interface {
	// Verify reports whether token is valid; a token is valid once. An
	// error means it could not be checked.
	Verify(ctx context.Context, token string) (bool, error)
}

// DemoOptions configures POST /v1/demo/check, the demo access of the
// site's checker.
type DemoOptions struct {
	// Origin is the site's origin, the only one CORS allows; empty allows
	// none.
	Origin string
	// Verifier checks the anti-abuse token of each request; nil answers
	// every request with a 503.
	Verifier ChallengeVerifier
	// Limit limits the demo checks of each client address; nil applies
	// config.DefaultRateLimitDemo a minute.
	Limit Limiter
	// GlobalLimit limits the demo checks of every client together; nil
	// applies config.DefaultRateLimitDemoGlobal a minute.
	GlobalLimit Limiter
}

// Names of the demo quota policies in the RateLimit fields.
const (
	demoPolicy       = "demo"
	demoGlobalPolicy = "demo_global"
)

// everyClient is the key of the global demo quota: one bucket for all.
const everyClient = ""

// demoMaxBytes bounds the body of a demo check: an input pasted by a
// person and a token of at most 2048 characters.
const demoMaxBytes = 8 << 10

// demoRequest is a request as the contract's DemoRequest schema describes
// it; unknown fields are ignored.
type demoRequest struct {
	batchItem
	Token string `json:"turnstile_token"`
}

// demoGate admits to the demo check the requests within the quota of their
// client address and proven by their anti-abuse token, then within the
// global quota: only a solved challenge spends it, so a flood of forged
// tokens cannot lock every visitor out. Nothing of the request is kept:
// its input is checked, answered with no-store, and forgotten.
type demoGate struct {
	verifier ChallengeVerifier
	logger   *slog.Logger
	ps       problems
	perAddr  *quota
	global   *quota
	clientOf func(*http.Request) string
}

// newDemoGate returns the gate opts.Demo configures, knowing each client
// by clientOf.
func newDemoGate(opts Options, ps problems, clientOf func(*http.Request) string) *demoGate {
	return &demoGate{
		verifier: opts.Demo.Verifier, logger: opts.Logger, ps: ps, clientOf: clientOf,
		perAddr: newQuota(limiterOr(opts.Demo.Limit, config.DefaultRateLimitDemo), demoPolicy, ps),
		global:  newQuota(limiterOr(opts.Demo.GlobalLimit, config.DefaultRateLimitDemoGlobal), demoGlobalPolicy, ps),
	}
}

// check answers POST /v1/demo/check: the report of the body's input, as
// GET /v1/check answers it, once the gate admits the request. A body is
// refused before its token is spent. Without a verifier, every request is
// answered 503 at once.
func (g *demoGate) check(c Checker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.verifier == nil {
			g.ps.write(w, r, challengeUnavailable)
			return
		}
		client := g.clientOf(r)
		if !g.perAddr.admit(w, r, client) {
			return
		}
		token, in, p, ok := demoInput(w, r)
		if !ok {
			g.ps.write(w, r, p)
			return
		}
		if p, ok := verifyChallenge(r.Context(), g.verifier, g.logger, token); !ok {
			g.ps.write(w, r, p)
			return
		}
		if !g.admitGlobally(w, r, client) {
			return
		}
		writeJSON(w, "application/json", http.StatusOK, wire.ReportOf(in.Kind(), c.Check(r.Context(), in)))
	})
}

// admitGlobally spends one request of the global quota and reports
// whether it allows it; otherwise it answers its 429 and gives the request
// back to the quota of client.
func (g *demoGate) admitGlobally(w http.ResponseWriter, r *http.Request, client string) bool {
	d := g.global.limiter.Allow(everyClient)
	if !d.Allowed {
		g.perAddr.limiter.Refund(client)
		g.global.deny(w, r, d)
	}
	return d.Allowed
}

// verifyChallenge returns the problem refusing token, and false, unless v
// accepts it. A token that cannot be checked fails closed, logged on
// logger.
func verifyChallenge(ctx context.Context, v ChallengeVerifier, logger *slog.Logger, token string) (problem, bool) {
	ok, err := v.Verify(ctx, token)
	switch {
	case err != nil:
		logger.ErrorContext(ctx, "challenge not checked", "error", err.Error())
		return challengeUnavailable, false
	case !ok:
		return challengeFailed, false
	}
	return problem{}, true
}

// demoInput returns the token the body of r holds and the input it asks
// to check, or the problem refusing it.
func demoInput(w http.ResponseWriter, r *http.Request) (string, domain.Input, problem, bool) {
	var req demoRequest
	if p, ok := decodeBody(w, r, demoMaxBytes, &req, invalidDemoBody); !ok {
		return "", nil, p, false
	}
	in, p, ok := itemInput(req.batchItem)
	return req.Token, in, p, ok
}
