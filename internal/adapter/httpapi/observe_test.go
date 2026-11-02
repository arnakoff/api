// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/adapter/httpapi/contracttest"
)

func TestAccessLogNamesTheRouteNeverThePathOrQuery(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name, target, route string
		status              float64
	}{
		{"matched", "/healthz?q=jean.dupont@exemple.fr", "GET /healthz", 200},
		{"unmatched", "/jean.dupont@exemple.fr", "", 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, logs := options(t, true)
			serve(t, httpapi.NewHandler(opts), http.MethodGet, tt.target, nil)

			accessLine(t, logs.Lines(t), map[string]any{"method": "GET", "route": tt.route, "status": tt.status})
			if strings.Contains(logs.String(), "dupont") {
				t.Errorf("log %s quotes the request's path or query", logs.String())
			}
		})
	}
}

// accessLine fails unless lines is one access line holding want.
func accessLine(t *testing.T, lines []map[string]any, want map[string]any) {
	t.Helper()
	if len(lines) != 1 {
		t.Fatalf("logged %d lines, want 1", len(lines))
	}
	if _, ok := lines[0]["duration"]; !ok || lines[0]["msg"] != "request" {
		t.Errorf("log = %v, want an access line with a duration", lines[0])
	}
	for k, v := range want {
		if lines[0][k] != v {
			t.Errorf("log %s = %v, want %v", k, lines[0][k], v)
		}
	}
}

func TestRoutesGetTheirPathWildcards(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	var id string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/items/{id}", func(_ http.ResponseWriter, r *http.Request) { id = r.PathValue("id") })
	serve(t, httpapi.Wrap(opts, httpapi.Route(opts, mux)), http.MethodGet, "/v1/items/42", nil)
	if id != "42" {
		t.Errorf("PathValue(id) = %q, want 42", id)
	}
}

// panicking is a handler that panics with a value holding personal data.
var panicking = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
	panic("lookup jean.dupont@exemple.fr failed")
})

func TestAPanicGivesAProblemAndAnErrorLine(t *testing.T) {
	t.Parallel()

	opts, logs := options(t, true)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/check?q=armen.ltd", nil)
	rec := httptest.NewRecorder()
	httpapi.Wrap(opts, panicking).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	c, err := contracttest.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Check(t.Context(), req, rec.Result()); err != nil {
		t.Errorf("the 500 breaks the contract: %v", err)
	}
	var levels []any
	for _, l := range logs.Lines(t) {
		levels = append(levels, l["level"])
	}
	if len(levels) != 2 || levels[0] != "ERROR" {
		t.Errorf("log levels = %v, want an ERROR line then the access line", levels)
	}
	if strings.Contains(logs.String(), "dupont") {
		t.Errorf("log %s quotes the panic value", logs.String())
	}
}

func TestAPanicAfterTheHeaderOnlyLogs(t *testing.T) {
	t.Parallel()

	opts, logs := options(t, true)
	late := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		panic("late")
	})
	res := serve(t, httpapi.Wrap(opts, late), http.MethodGet, "/v1/check", nil)
	if res.Code != http.StatusAccepted {
		t.Errorf("status = %d, want the 202 already sent", res.Code)
	}
	if lines := logs.Lines(t); len(lines) != 2 {
		t.Errorf("logged %d lines, want the error and the access line", len(lines))
	}
}

func TestAnAbortedHandlerIsNotRecovered(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	abort := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
	defer func() {
		if v := recover(); !errors.Is(v.(error), http.ErrAbortHandler) {
			t.Errorf("recovered %v, want http.ErrAbortHandler re-panicked", v)
		}
	}()
	serve(t, httpapi.Wrap(opts, abort), http.MethodGet, "/v1/check", nil)
	t.Error("ErrAbortHandler was swallowed")
}
