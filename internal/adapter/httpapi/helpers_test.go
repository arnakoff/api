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
	"github.com/arnakoff/api/internal/platform/config"
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

func (c *checker) CheckAll(ctx context.Context, ins []domain.Input) []domain.Report {
	return checkEach(ctx, c.Check, ins)
}

// checkEach returns the report check gives of each input, one after the
// other: the CheckAll of a fake.
func checkEach(ctx context.Context, check func(context.Context, domain.Input) domain.Report, ins []domain.Input) []domain.Report {
	reports := make([]domain.Report, len(ins))
	for i, in := range ins {
		reports[i] = check(ctx, in)
	}
	return reports
}

// inputs returns the inputs c was asked about.
func (c *checker) inputs() []domain.Input {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]domain.Input(nil), c.asked...)
}

// testKey is the API key the fake authenticator of options accepts. Its
// format is not checked: the fake does not parse it.
const testKey = "ark_test_000000000000000000000000000000abcdef"

// keyring is a fake authenticator: it answers err, or accepts the key when
// err is nil.
type keyring struct {
	err error
}

func (k keyring) Authenticate(context.Context, string) (domain.KeyRecord, error) {
	if k.err != nil {
		return domain.KeyRecord{}, k.err
	}
	return domain.KeyRecord{ID: 1, Environment: domain.KeyTest}, nil
}

// authorize sets the Authorization field of req to testKey.
func authorize(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer "+testKey)
	return req
}

func options(t *testing.T, ready bool) (httpapi.Options, *loggingtest.Buffer) {
	t.Helper()
	logger, buf := loggingtest.New(t)
	return httpapi.Options{
		Ready: readiness(ready), Check: &checker{}, Auth: keyring{}, Logger: logger, NewID: func() string { return "GENERATED" },
		ProblemBase: problemBase, BatchMaxItems: config.DefaultBatchMaxItems,
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
