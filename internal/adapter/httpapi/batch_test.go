// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

// batchCall is a POST /v1/check/batch to send.
type batchCall struct {
	body        string
	contentType string // "application/json" when empty, no header when "none"
	maxItems    int    // Options.BatchMaxItems, the default when zero
	refused     bool   // the contract refuses the request
}

// send posts call to a handler whose check answers report, holds the
// exchange to the contract and returns the response and the fake.
func (call batchCall) send(t *testing.T, report domain.Report) (*httptest.ResponseRecorder, *checker) {
	t.Helper()
	opts, _ := options(t, true)
	c := &checker{report: report}
	opts.Check = c
	if call.maxItems != 0 {
		opts.BatchMaxItems = call.maxItems
	}
	rec := httptest.NewRecorder()
	httpapi.NewHandler(opts).ServeHTTP(rec, call.request(t))
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	hold := ct.Check
	if call.refused {
		hold = ct.CheckRefused
	}
	if err := hold(t.Context(), call.request(t), rec.Result()); err != nil {
		t.Errorf("POST %.60s breaks the contract: %v", call.body, err)
	}
	return rec, c
}

// request returns a new request of call: the handler and the contract each
// read a body.
func (call batchCall) request(t *testing.T) *http.Request {
	t.Helper()
	req := authorize(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/check/batch", strings.NewReader(call.body)))
	switch call.contentType {
	case "":
		req.Header.Set("Content-Type", "application/json")
	case "none":
	default:
		req.Header.Set("Content-Type", call.contentType)
	}
	return req
}

// results returns the results of a batch response.
func results(t *testing.T, res *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	raw, ok := decode(t, res)["results"].([]any)
	if !ok {
		t.Fatal("no results in the body")
	}
	out := make([]map[string]any, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]any)
	}
	return out
}

// items returns a batch body of n copies of item.
func items(n int, item string) string {
	return `{"items":[` + strings.TrimSuffix(strings.Repeat(item+",", n), ",") + `]}`
}

func TestBatchAnswersAResultPerItemInOrder(t *testing.T) {
	t.Parallel()

	res, c := batchCall{body: `{"items":[{"q":"armen.ltd"},{"q":"FR76 0000","type":"iban"},{"q":"552100554","type":"siren"}]}`}.
		send(t, blacklisted())

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	got := results(t, res)
	if len(got) != 3 {
		t.Fatalf("%d results, want 3", len(got))
	}
	for _, i := range []int{0, 2} {
		if report, _ := got[i]["report"].(map[string]any); report["input"] != "armen.ltd" {
			t.Errorf("result %d = %v, want the report", i, got[i])
		}
	}
	e, _ := got[1]["error"].(map[string]any)
	if e["code"] != "invalid_input" || e["detail"] != "L'entrée n'est pas un IBAN valide." {
		t.Errorf("result 1 = %v, want the invalid IBAN problem", got[1])
	}
	asked := c.inputs()
	if len(asked) != 2 || asked[0].String() != "armen.ltd" || asked[1].Kind() != domain.KindSIREN {
		t.Errorf("checked %v, want armen.ltd then the SIREN", asked)
	}
}

func TestBatchReportsTheTypeOfEachItem(t *testing.T) {
	t.Parallel()

	res, _ := batchCall{body: `{"items":[{"q":"552100554"}]}`}.send(t, blacklisted())

	if report, _ := results(t, res)[0]["report"].(map[string]any); report["type"] != "siren" {
		t.Errorf("type = %v, want siren", report["type"])
	}
}

func TestBatchRefusesAnItemAloneWithoutFailingTheBatch(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name, item, code string
		refused          bool
	}{
		{"without q", `{"type":"domain"}`, "missing_parameter", true},
		{"with an empty q", `{"q":""}`, "missing_parameter", true},
		{"with an unknown type", `{"q":"armen.ltd","type":"url"}`, "invalid_parameter", true},
		{"of no kind", `{"q":"a b"}`, "invalid_input", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, c := batchCall{body: `{"items":[` + tt.item + `,{"q":"armen.ltd"}]}`, refused: tt.refused}.
				send(t, blacklisted())

			got := results(t, res)
			if e, _ := got[0]["error"].(map[string]any); e["code"] != tt.code || e["instance"] != nil {
				t.Errorf("result 0 = %v, want a %s problem without instance", got[0], tt.code)
			}
			if _, ok := got[1]["report"]; !ok {
				t.Errorf("result 1 = %v, want the report", got[1])
			}
			if asked := c.inputs(); len(asked) != 1 {
				t.Errorf("checked %v, want armen.ltd only", asked)
			}
		})
	}
}

func TestBatchRefusesMoreItemsThanTheLimit(t *testing.T) {
	t.Parallel()

	for _, limit := range []int{2, config.DefaultBatchMaxItems} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			t.Parallel()
			res, c := batchCall{body: items(limit+1, `{"q":"armen.ltd"}`), maxItems: limit}.send(t, blacklisted())

			got := decode(t, res)
			want := fmt.Sprintf("Le lot dépasse le nombre maximal d'éléments autorisé : %d.", limit)
			if res.Code != http.StatusUnprocessableEntity || got["code"] != "too_many_items" || got["detail"] != want {
				t.Errorf("answer = %d %v, want 422 too_many_items naming %d", res.Code, got, limit)
			}
			if asked := c.inputs(); len(asked) != 0 {
				t.Errorf("checked %d inputs, want no check", len(asked))
			}
		})
	}
}

func TestBatchChecksAsManyItemsAsTheLimit(t *testing.T) {
	t.Parallel()

	res, c := batchCall{body: items(2, `{"q":"armen.ltd"}`), maxItems: 2}.send(t, blacklisted())

	if res.Code != http.StatusOK || len(c.inputs()) != 2 {
		t.Errorf("answer = %d after %d checks, want 200 after 2", res.Code, len(c.inputs()))
	}
}

func TestBatchIsNeverCached(t *testing.T) {
	t.Parallel()

	res, _ := batchCall{body: `{"items":[{"q":"armen.ltd"}]}`}.send(t, blacklisted())

	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store: the inputs may be personal data", got)
	}
}

// BenchmarkCheckBatch measures POST /v1/check/batch of a full batch through
// the whole handler, around a check that answers at once: the cost the API
// adds to the checks themselves.
func BenchmarkCheckBatch(b *testing.B) {
	h := httpapi.NewHandler(httpapi.Options{
		Ready: readiness(true), Check: &benchChecker{report: blacklisted()}, Auth: keyring{},
		Logger: slog.New(slog.DiscardHandler), ProblemBase: problemBase, BatchMaxItems: config.DefaultBatchMaxItems,
		KeyLimit: frozenLimiter(1 << 30), IPLimit: frozenLimiter(1 << 30), AuthLimit: frozenLimiter(1 << 30),
	})
	body := items(config.DefaultBatchMaxItems, `{"q":"armen.ltd"}`)
	b.ReportAllocs()
	for b.Loop() {
		req := authorize(httptest.NewRequestWithContext(b.Context(), http.MethodPost, "/v1/check/batch", strings.NewReader(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d", rec.Code)
		}
	}
}

func TestBatchSpendsOneUnitOfTheKeyQuotaPerItem(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	opts.Check = &checker{report: blacklisted()}
	opts.KeyLimit = frozenLimiter(10)
	h := httpapi.NewHandler(opts)
	post := func(n int) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, batchCall{body: items(n, `{"q":"armen.ltd"}`)}.request(t))
		return rec
	}

	if res := post(4); res.Code != http.StatusOK || res.Header().Get("RateLimit") != `"key";r=6;t=6` {
		t.Fatalf("batch of 4 = %d, RateLimit %q; want 200 and 6 left", res.Code, res.Header().Get("RateLimit"))
	}
	res := post(7)
	if res.Code != http.StatusTooManyRequests || res.Header().Get("Retry-After") != "6" || res.Header().Get("RateLimit") != `"key";r=6;t=6` {
		t.Errorf("batch of 7 with 6 left = %d, Retry-After %q, RateLimit %q; want 429, 6 and 6 left",
			res.Code, res.Header().Get("Retry-After"), res.Header().Get("RateLimit"))
	}
	if res := post(6); res.Code != http.StatusOK {
		t.Errorf("batch of 6 after the refused one = %d, want 200: a refused batch spends nothing", res.Code)
	}
}
