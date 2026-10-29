// SPDX-License-Identifier: AGPL-3.0-only

package logging

import (
	"context"
	"log/slog"
	"slices"
)

// RequestIDKey is the key of the request identifier in a log line.
const RequestIDKey = "request_id"

// requestIDContextKey is the context key of the request identifier.
type requestIDContextKey struct{}

// WithRequestID returns a copy of ctx carrying the request identifier id,
// which every line logged with that context then holds.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

// RequestID returns the request identifier ctx carries. It reports false
// when there is none or it is empty.
func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDContextKey{}).(string)
	return id, ok && id != ""
}

// requestIDHandler adds the request identifier of the context to every
// record, at the top level of the line even under a group.
type requestIDHandler struct {
	// root is the wrapped handler before any With call; current is root
	// with steps applied, in order.
	root    slog.Handler
	current slog.Handler
	steps   []func(slog.Handler) slog.Handler
	grouped bool
}

// newRequestIDHandler wraps h.
func newRequestIDHandler(h slog.Handler) *requestIDHandler {
	return &requestIDHandler{root: h, current: h}
}

// Enabled reports whether the wrapped handler handles level.
func (h *requestIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.current.Enabled(ctx, level)
}

// Handle adds the request identifier of ctx, if any, and hands r on.
func (h *requestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	id, ok := RequestID(ctx)
	if !ok {
		return h.current.Handle(ctx, r)
	}
	attr := slog.String(RequestIDKey, id)
	if !h.grouped {
		r.AddAttrs(attr)
		return h.current.Handle(ctx, r)
	}
	// Record attributes land in the open groups: replay the With calls on
	// top of the identifier instead.
	next := h.root.WithAttrs([]slog.Attr{attr})
	for _, step := range h.steps {
		next = step(next)
	}
	return next.Handle(ctx, r)
}

// WithAttrs returns a handler adding attrs to every record.
func (h *requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.with(func(next slog.Handler) slog.Handler { return next.WithAttrs(attrs) }, false)
}

// WithGroup returns a handler qualifying later attributes by name.
func (h *requestIDHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.with(func(next slog.Handler) slog.Handler { return next.WithGroup(name) }, true)
}

// with returns a copy of h with step applied; group tells whether step
// opens a group.
func (h *requestIDHandler) with(step func(slog.Handler) slog.Handler, group bool) *requestIDHandler {
	return &requestIDHandler{
		root:    h.root,
		current: step(h.current),
		steps:   append(slices.Clip(h.steps), step),
		grouped: h.grouped || group,
	}
}
