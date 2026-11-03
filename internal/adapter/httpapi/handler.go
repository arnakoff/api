// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/netip"
	"path"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

// Readiness tells whether the API can answer checks.
type Readiness interface {
	// Ready reports whether every snapshot is loaded.
	Ready() bool
}

// Checker checks an input against the sources.
type Checker interface {
	// Check returns the report of in; a source it could not consult is
	// reported unavailable, never an error.
	Check(ctx context.Context, in domain.Input) domain.Report
	// CheckAll returns the report of each input, in the order of ins.
	CheckAll(ctx context.Context, ins []domain.Input) []domain.Report
}

// Options configures the handler.
type Options struct {
	// Ready backs /readyz.
	Ready Readiness
	// Check backs /v1/check and /v1/check/batch.
	Check Checker
	// Auth checks the API key of /v1/check and /v1/check/batch; nil
	// answers them all with a 503.
	Auth Authenticator
	// Sources backs /v1/sources.
	Sources SourceLister
	// Logger receives the access log and the recovered panics.
	Logger *slog.Logger
	// NewID makes a request identifier when the client sent no valid one;
	// nil means crypto/rand.Text.
	NewID func() string
	// ProblemBase is the URL of the documentation of the problems, ending
	// with a slash: the type of a problem is its code under it.
	ProblemBase string
	// BatchMaxItems is the most items POST /v1/check/batch accepts, at
	// least one: config.Config.BatchMaxItems.
	BatchMaxItems int
	// KeyLimit limits the requests of each API key to the check routes;
	// nil applies config.DefaultRateLimitKey a minute.
	KeyLimit Limiter
	// IPLimit limits the requests of each client address to the routes
	// needing no key; nil applies config.DefaultRateLimitIP a minute.
	IPLimit Limiter
	// AuthLimit limits the refused API keys of each client address on the
	// check routes: an address it refuses is answered 429 before its key is
	// looked up. nil applies config.DefaultRateLimitAuth a minute.
	AuthLimit RefusalLimiter
	// TrustedProxies are the proxies whose X-Forwarded-For field tells the
	// client's address: config.Config.TrustedProxies.
	TrustedProxies []netip.Prefix
	// Demo configures POST /v1/demo/check.
	Demo DemoOptions
	// MCP serves the remote MCP endpoint, POST /mcp, behind the API key
	// and the quota of the check routes; nil leaves the route out.
	MCP http.Handler
}

// NewHandler returns the handler of the API: its routes behind the request
// middleware.
func NewHandler(opts Options) http.Handler {
	ps := problemsOf(opts)
	mux := http.NewServeMux()
	mux.Handle("GET "+healthzPath, healthz())
	mux.Handle("GET /readyz", readyz(opts.Ready))
	auth := opts.Auth
	if auth == nil {
		auth = noAuthenticator{}
	}
	byAddr := func(r *http.Request) string { return clientKey(r, opts.TrustedProxies) }
	gate := &keyGate{
		auth: auth, logger: opts.Logger, ps: ps, clientOf: byAddr,
		perKey:   newQuota(limiterOr(opts.KeyLimit, config.DefaultRateLimitKey), keyPolicy, ps),
		refusals: newRefusalBudget(refusalLimiterOr(opts.AuthLimit, config.DefaultRateLimitAuth), ps),
	}
	perAddr := newQuota(limiterOr(opts.IPLimit, config.DefaultRateLimitIP), ipPolicy, ps)
	anonymous := func(h http.Handler) http.Handler { return limit(perAddr, byAddr, h) }
	mux.Handle("GET /v1/check", gate.require(check(opts.Check, ps)))
	mux.Handle("POST /v1/check/batch", gate.require(batch(opts.Check, ps, opts.BatchMaxItems)))
	if opts.MCP != nil {
		mux.Handle("POST /mcp", gate.require(opts.MCP))
	}
	demo := newDemoGate(opts, ps, byAddr)
	mux.Handle("POST /v1/demo/check", allowOrigin(opts.Demo.Origin, demo.check(opts.Check)))
	mux.Handle("OPTIONS /v1/demo/check", preflight(opts.Demo.Origin))
	mux.Handle("GET /v1/sources", anonymous(sources(opts.Sources)))
	mux.Handle("GET /v1/signals", anonymous(signals()))
	return wrap(opts, route(ps, mux))
}

// wrap applies the request middleware to next, outermost first: request
// identifier, then access log and panic recovery, then security headers.
func wrap(opts Options, next http.Handler) http.Handler {
	newID := opts.NewID
	if newID == nil {
		newID = rand.Text
	}
	return withRequestID(newID, observe(opts.Logger, problemsOf(opts), secure(next)))
}

// secure sets the headers every response carries. A handler may replace
// Cache-Control for a response that can be cached.
func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// route dispatches to mux, but answers a problem where mux would answer a
// text error or redirect to the canonical path with an HTML body: the API
// redirects nothing.
func route(ps problems, mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if path.Clean(r.URL.Path) != r.URL.Path {
			ps.write(w, r, notFound)
			return
		}
		h, pattern := mux.Handler(r)
		if pattern != "" {
			// mux sets r.Pattern, which the access log names, and the
			// path wildcards: mux.Handler sets neither.
			mux.ServeHTTP(w, r)
			return
		}
		// Let mux tell 404 from 405 and set Allow, then discard its body.
		probe := &statusOnly{header: w.Header()}
		h.ServeHTTP(probe, r)
		if probe.status == http.StatusMethodNotAllowed {
			ps.write(w, r, methodNotAllowed)
			return
		}
		ps.write(w, r, notFound)
	})
}

// statusOnly is a ResponseWriter keeping the status and the headers of a
// response and dropping its body.
type statusOnly struct {
	header http.Header
	status int
}

func (s *statusOnly) Header() http.Header         { return s.header }
func (s *statusOnly) Write(b []byte) (int, error) { return len(b), nil }
func (s *statusOnly) WriteHeader(status int)      { s.status = status }
