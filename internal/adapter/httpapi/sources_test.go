// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// sourceLister is a fake list of sources.
type sourceLister []app.ListedSource

func (l sourceLister) Sources() []app.ListedSource { return l }

// amfListed is the source of the contract's SourceList example, fetched in
// another zone than UTC.
var amfListed = app.ListedSource{
	Info: domain.SourceInfo{
		Name: "AMF", Publisher: "Autorité des marchés financiers", Licence: "Licence Ouverte 2.0",
		URL: "https://www.data.gouv.fr/datasets/641d5dab1d84b530f7720b08",
	},
	Synced: true,
	Sync:   app.SourceSync{Source: "AMF", LastSuccess: time.Date(2026, 11, 2, 9, 14, 42, 0, paris), Rows: 3466},
}

// sources sends GET /v1/sources to a handler listing l and holds the
// exchange to the contract.
func sources(t *testing.T, l sourceLister) *httptest.ResponseRecorder {
	t.Helper()
	opts, _ := options(t, true)
	opts.Sources = l
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/sources", nil)
	rec := httptest.NewRecorder()
	httpapi.NewHandler(opts).ServeHTTP(rec, req)
	ct, err := contract()
	if err != nil {
		t.Fatal(err)
	}
	if err := ct.Check(t.Context(), req, rec.Result()); err != nil {
		t.Errorf("GET /v1/sources breaks the contract: %v", err)
	}
	return rec
}

func TestSourcesAnswersEachSourceAndTheStateOfItsData(t *testing.T) {
	t.Parallel()

	empty := app.ListedSource{
		Info:   domain.SourceInfo{Name: "ARCEP", Publisher: "ARCEP", Licence: "Licence Ouverte 1.0", URL: "https://arcep.example"},
		Synced: true,
		Sync:   app.SourceSync{Source: "ARCEP", LastFailure: time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)},
	}
	registry := app.ListedSource{
		Info: domain.SourceInfo{Name: "SIRENE", Publisher: "DINUM", Licence: "Licence Ouverte 2.0", URL: "https://sirene.example"},
	}

	res := sources(t, sourceLister{amfListed, empty, registry})

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	want := `{"sources":[` +
		`{"name":"AMF","publisher":"Autorité des marchés financiers","licence":"Licence Ouverte 2.0",` +
		`"url":"https://www.data.gouv.fr/datasets/641d5dab1d84b530f7720b08","status":"ok",` +
		`"updated_at":"2026-11-02T08:14:42Z","entries":3466},` +
		`{"name":"ARCEP","publisher":"ARCEP","licence":"Licence Ouverte 1.0","url":"https://arcep.example",` +
		`"status":"indisponible","updated_at":null,"entries":0},` +
		`{"name":"SIRENE","publisher":"DINUM","licence":"Licence Ouverte 2.0","url":"https://sirene.example",` +
		`"status":"ok","updated_at":null,"entries":null}]}` + "\n"
	if got := res.Body.String(); got != want {
		t.Errorf("body =\n%s\nwant\n%s", got, want)
	}
}

func TestSourcesAnswersAnEmptyListAsAnArray(t *testing.T) {
	t.Parallel()

	res := sources(t, sourceLister{})

	if got, want := res.Body.String(), `{"sources":[]}`+"\n"; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}
