// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"path"

	"github.com/arnakoff/api/internal/domain"
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
}

// NewHandler returns the handler of the API: its routes behind the request
// middleware.
func NewHandler(opts Options) http.Handler {
	ps := problemsOf(opts)
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz())
	mux.Handle("GET /readyz", readyz(opts.Ready))
	mux.Handle("GET /v1/check", check(opts.Check, ps))
	mux.Handle("POST /v1/check/batch", batch(opts.Check, ps, opts.BatchMaxItems))
	mux.Handle("GET /v1/sources", sources(opts.Sources))
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
