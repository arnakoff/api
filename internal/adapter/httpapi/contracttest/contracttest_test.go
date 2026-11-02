// SPDX-License-Identifier: AGPL-3.0-only

package contracttest_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/arnakoff/api/api"
	"github.com/arnakoff/api/internal/adapter/httpapi/contracttest"
)

// spec is the contract, parsed once for every test.
var spec = sync.OnceValues(func() (*openapi3.T, error) {
	return openapi3.NewLoader().LoadFromData(api.Spec())
})

// contract is the harness under test, loaded once: it is read-only.
var contract = sync.OnceValues(contracttest.Load)

func load(t *testing.T) *contracttest.Contract {
	t.Helper()
	c, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// example returns a fresh copy of the value of the contract's example name,
// as a JSON object: the contract's own examples are the real responses under
// test.
func example(t *testing.T, name string) map[string]any {
	t.Helper()
	doc, err := spec()
	if err != nil {
		t.Fatal(err)
	}
	ref, ok := doc.Components.Examples[name]
	if !ok {
		t.Fatalf("no example %q in the contract", name)
	}
	b, err := json.Marshal(ref.Value.Value)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func firstSignal(v map[string]any) map[string]any {
	return v["signals"].([]any)[0].(map[string]any)
}

// exchange is one request and the response the contract is checked on. An
// empty method, status or header is GET, 200 and JSON. The request carries
// an API key unless anonymous. A refused exchange is checked with
// CheckRefused.
type exchange struct {
	method, target, reqBody string
	status                  int
	header                  http.Header
	body                    any
	anonymous, refused      bool
}

func (e exchange) check(t *testing.T) error {
	t.Helper()
	if e.method == "" {
		e.method = http.MethodGet
	}
	if e.status == 0 {
		e.status = http.StatusOK
	}
	if e.header == nil {
		e.header = jsonHeader("application/json")
	}
	req := httptest.NewRequestWithContext(t.Context(), e.method, e.target, strings.NewReader(e.reqBody))
	if e.reqBody != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if !e.anonymous {
		req.Header.Set("Authorization", "Bearer ark_test_000000000000000000000000000000abcdef")
	}
	rec := httptest.NewRecorder()
	for k, vs := range e.header {
		rec.Header()[k] = vs
	}
	rec.WriteHeader(e.status)
	if err := json.NewEncoder(rec).Encode(e.body); err != nil {
		t.Fatal(err)
	}
	if e.refused {
		return load(t).CheckRefused(t.Context(), req, rec.Result())
	}
	return load(t).Check(t.Context(), req, rec.Result())
}

func jsonHeader(ct string) http.Header { return http.Header{"Content-Type": {ct}} }

// challenged returns the header of a 401 problem with the WWW-Authenticate
// challenge c.
func challenged(c string) http.Header {
	h := jsonHeader("application/problem+json")
	h.Set("WWW-Authenticate", c)
	return h
}

func TestCheckAcceptsResponsesThatFollowTheContract(t *testing.T) {
	t.Parallel()

	limited := jsonHeader("application/problem+json")
	limited.Set("RateLimit", `"default";r=0;t=30`)
	limited.Set("RateLimit-Policy", `"default";q=100;w=60`)
	limited.Set("Retry-After", "30")
	problem := jsonHeader("application/problem+json")
	batch := `{"items":[{"q":"armen.ltd"},{"q":"FR76 0000","type":"iban"}]}`
	for name, e := range map[string]exchange{
		"report":             {target: "/v1/check?q=armen.ltd", body: example(t, "ReportBlacklisted")},
		"report with advice": {target: "/v1/check?q=ameli-remboursement.com&type=domain", body: example(t, "ReportLookalike")},
		"no signal":          {target: "/v1/check?q=552100554", body: example(t, "ReportNoSignal")},
		"problem":            {target: "/v1/check?q=x", status: http.StatusUnprocessableEntity, header: problem, body: example(t, "ProblemInvalidInput")},
		"rate limited":       {target: "/v1/check?q=armen.ltd", status: http.StatusTooManyRequests, header: limited, body: example(t, "ProblemInvalidInput")},
		"batch":              {method: http.MethodPost, target: "/v1/check/batch", reqBody: batch, body: example(t, "BatchResponseMixed")},
		"open request":       {method: http.MethodPost, target: "/v1/check/batch", reqBody: `{"items":[{"q":"armen.ltd","note":"x"}],"x":1}`, body: example(t, "BatchResponseMixed")},
		"sources":            {target: "/v1/sources", body: example(t, "SourceList")},
		"anonymous sources":  {target: "/v1/sources", body: example(t, "SourceList"), anonymous: true},
		"invalid key":        {target: "/v1/check?q=armen.ltd", status: http.StatusUnauthorized, header: challenged(`Bearer realm="arnakoff", error="invalid_token"`), body: example(t, "ProblemInvalidApiKey")},
		"key unchecked":      {target: "/v1/check?q=armen.ltd", status: http.StatusServiceUnavailable, header: problem, body: example(t, "ProblemAuthenticationUnavailable")},
		"signals":            {target: "/v1/signals", body: example(t, "SignalCatalog")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := e.check(t); err != nil {
				t.Errorf("Check: %v", err)
			}
		})
	}
}

func TestCheckRejectsResponsesThatBreakTheContract(t *testing.T) {
	t.Parallel()

	withoutSourceURL := example(t, "ReportBlacklisted")
	delete(firstSignal(withoutSourceURL), "source_url")
	undocumentedField := example(t, "ReportBlacklisted")
	undocumentedField["verdict"] = "sûr"
	undocumentedSignalField := example(t, "ReportBlacklisted")
	firstSignal(undocumentedSignalField)["score"] = 1
	spacedStatus := example(t, "ReportNoSignal")
	spacedStatus["sources"].([]any)[0].(map[string]any)["status"] = "non applicable"
	report := example(t, "ReportBlacklisted")

	for name, e := range map[string]exchange{
		"missing required field":         {target: "/v1/check?q=x", body: withoutSourceURL},
		"undocumented field":             {target: "/v1/check?q=x", body: undocumentedField},
		"undocumented nested field":      {target: "/v1/check?q=x", body: undocumentedSignalField},
		"value outside the enumeration":  {target: "/v1/check?q=x", body: spacedStatus},
		"undocumented status":            {target: "/v1/check?q=x", status: http.StatusTeapot, body: report},
		"undocumented content type":      {target: "/v1/check?q=x", header: jsonHeader("text/plain"), body: report},
		"rate limited without headers":   {target: "/v1/check?q=x", status: http.StatusTooManyRequests, header: jsonHeader("application/problem+json"), body: example(t, "ProblemInvalidInput")},
		"request with an unknown type":   {target: "/v1/check?q=x&type=url", body: report},
		"request without required param": {target: "/v1/check", body: report},
		"batch request without items":    {method: http.MethodPost, target: "/v1/check/batch", reqBody: `{"items":[]}`, body: example(t, "BatchResponseMixed")},
		"request without a key":          {target: "/v1/check?q=armen.ltd", body: report, anonymous: true},
		"401 without a challenge":        {target: "/v1/check?q=armen.ltd", status: http.StatusUnauthorized, header: jsonHeader("application/problem+json"), body: example(t, "ProblemInvalidApiKey")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := e.check(t); err == nil {
				t.Error("Check = nil, want an error")
			}
		})
	}
}

func TestCheckRejectsAnUndocumentedOperation(t *testing.T) {
	t.Parallel()

	for _, e := range []exchange{
		{target: "/v1/unknown", body: map[string]any{}},
		{method: http.MethodDelete, target: "/v1/check", body: map[string]any{}},
	} {
		if err := e.check(t); !errors.Is(err, contracttest.ErrUndocumented) {
			t.Errorf("%s %s: Check = %v, want ErrUndocumented", e.method, e.target, err)
		}
	}
}

func TestCheckRefusedHoldsTheAnswerToARequestTheContractRefuses(t *testing.T) {
	t.Parallel()

	problem := jsonHeader("application/problem+json")
	missing := exchange{
		target: "/v1/check", status: http.StatusBadRequest, header: problem,
		body: example(t, "ProblemMissingParameter"), refused: true,
	}
	if err := missing.check(t); err != nil {
		t.Errorf("CheckRefused of a documented problem: %v", err)
	}
	noKey := exchange{
		target: "/v1/check?q=armen.ltd", status: http.StatusUnauthorized, header: challenged(`Bearer realm="arnakoff"`),
		body: example(t, "ProblemMissingApiKey"), anonymous: true, refused: true,
	}
	if err := noKey.check(t); err != nil {
		t.Errorf("CheckRefused of a request without a key: %v", err)
	}
	for name, e := range map[string]exchange{
		"undocumented status": {target: "/v1/check?q=x&type=url", status: http.StatusTeapot, header: problem, body: example(t, "ProblemInvalidParameter"), refused: true},
		"report to a refusal": {target: "/v1/check?type=domain", status: http.StatusBadRequest, header: problem, body: example(t, "ReportBlacklisted"), refused: true},
		"undocumented":        {target: "/v1/unknown", body: map[string]any{}, refused: true},
	} {
		if err := e.check(t); err == nil {
			t.Errorf("%s: CheckRefused = nil, want an error", name)
		}
	}
}

func TestCheckRefusedRejectsAValidRequest(t *testing.T) {
	t.Parallel()

	e := exchange{target: "/v1/check?q=armen.ltd", body: example(t, "ReportBlacklisted"), refused: true}
	if err := e.check(t); !errors.Is(err, contracttest.ErrRequestValid) {
		t.Errorf("CheckRefused = %v, want ErrRequestValid", err)
	}
}
