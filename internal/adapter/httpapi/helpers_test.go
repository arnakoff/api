// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/logging/loggingtest"
)

// problemBase is the documentation of the problems in the handler tests.
const problemBase = "https://arnakoff.com/docs/erreurs/"

// readiness is a fake readiness.
type readiness bool

func (r readiness) Ready() bool { return bool(r) }

// checker is a fake check: it answers report, whatever the input, and
// records the inputs it was asked about.
type checker struct {
	report domain.Report
	mu     sync.Mutex
	asked  []domain.Input
}

func (c *checker) Check(_ context.Context, in domain.Input) domain.Report {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked = append(c.asked, in)
	return c.report
}

// inputs returns the inputs c was asked about.
func (c *checker) inputs() []domain.Input {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]domain.Input(nil), c.asked...)
}

func options(t *testing.T, ready bool) (httpapi.Options, *loggingtest.Buffer) {
	t.Helper()
	logger, buf := loggingtest.New(t)
	return httpapi.Options{
		Ready: readiness(ready), Check: &checker{}, Logger: logger, NewID: func() string { return "GENERATED" },
		ProblemBase: problemBase,
	}, buf
}

// serve sends a request to h and returns the recorded response.
func serve(t *testing.T, h http.Handler, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	for k, vs := range header {
		req.Header[k] = vs
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decode reads the JSON body of res.
func decode(t *testing.T, res *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatalf("decode the body: %v", err)
	}
	return v
}

func hasNoHTML(t *testing.T, res *httptest.ResponseRecorder) {
	t.Helper()
	if ct := res.Header().Get("Content-Type"); strings.Contains(ct, "html") {
		t.Errorf("Content-Type = %q, want no HTML", ct)
	}
}
