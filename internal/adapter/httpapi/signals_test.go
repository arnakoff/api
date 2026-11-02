// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/domain/signal"
)

// signalDefinition is a definition as the contract's SignalDefinition
// describes it.
type signalDefinition struct {
	Code      string   `json:"code"`
	Level     string   `json:"level"`
	Inputs    []string `json:"inputs"`
	Source    string   `json:"source"`
	Condition string   `json:"condition"`
}

// listSignals sends GET /v1/signals to the handler and holds the exchange
// to the contract.
func listSignals(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	opts, _ := options(t, true)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/signals", nil)
	rec := httptest.NewRecorder()
	httpapi.NewHandler(opts).ServeHTTP(rec, req)
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	if err := ct.Check(t.Context(), req, rec.Result()); err != nil {
		t.Errorf("GET /v1/signals breaks the contract: %v", err)
	}
	return rec
}

func TestSignalsAnswersTheCatalog(t *testing.T) {
	t.Parallel()

	res := listSignals(t)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body struct {
		Signals []signalDefinition `json:"signals"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	catalog := signal.Catalog()
	if len(body.Signals) != len(catalog) {
		t.Fatalf("got %d signals, want the %d of the catalog", len(body.Signals), len(catalog))
	}
	want := signalDefinition{
		Code: "amf_blacklist", Level: "alerte", Inputs: []string{"domain", "email"}, Source: "AMF",
		Condition: "Site inscrit ou adresse e-mail inscrite à l'identique",
	}
	if got := body.Signals[0]; got.Code != want.Code || got.Level != want.Level || got.Source != want.Source ||
		strings.Join(got.Inputs, ",") != strings.Join(want.Inputs, ",") || got.Condition != want.Condition {
		t.Errorf("signals[0] = %+v, want %+v", got, want)
	}
	for i, d := range catalog {
		if got := body.Signals[i]; got.Code != d.Code || got.Source != string(d.Source) {
			t.Errorf("signals[%d] = %s from %s, want %s from %s", i, got.Code, got.Source, d.Code, d.Source)
		}
	}
}
