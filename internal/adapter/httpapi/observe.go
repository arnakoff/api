// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// observe logs one access line per request and turns a panic into a 500
// problem. The line holds the path, never the query: the query holds the
// checked input, personal data for an email, a phone number or an IBAN. A
// panic is logged with its type and stack, never its value, which may
// quote the input.
func observe(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recorder{ResponseWriter: w}
		start := time.Now()
		defer func() { finish(logger, r, rec, start, recover()) }()
		next.ServeHTTP(rec, r)
	})
}

// finish ends the request r: it recovers the panic v, if any, and logs the
// access line. http.ErrAbortHandler is panicked again, as net/http expects.
func finish(logger *slog.Logger, r *http.Request, rec *recorder, start time.Time, v any) {
	ctx := r.Context()
	if v != nil {
		if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
			panic(v)
		}
		logger.ErrorContext(ctx, "panic serving a request",
			"panic", fmt.Sprintf("%T", v), "stack", string(debug.Stack()))
		if rec.status == 0 {
			writeProblem(rec, internalError)
		}
	}
	status := rec.status
	if status == 0 {
		status = http.StatusOK // net/http sends 200 for a handler that wrote nothing
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "request",
		slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", status),
		slog.Int("bytes", rec.bytes), slog.Duration("duration", time.Since(start)))
}

// recorder is a ResponseWriter remembering the status and the size of the
// response. status is 0 until the header is sent.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

// WriteHeader records status and sends it.
func (r *recorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

// Write sends b, with a 200 status if none was sent.
func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Unwrap gives http.ResponseController the underlying writer.
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
