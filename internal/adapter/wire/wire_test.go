// SPDX-License-Identifier: AGPL-3.0-only

package wire_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// paris is a zone other than UTC: times are written in UTC.
var paris = time.FixedZone("CET", 3600)

// marshal returns the JSON of v.
func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestReportOfWritesTheReportInUTC(t *testing.T) {
	t.Parallel()

	r := domain.NewReport("armen.ltd", time.Date(2026, 11, 2, 9, 25, 53, 0, paris), []domain.Signal{{
		Code: "amf_blacklist", Level: domain.LevelAlert, Message: "Figure sur la liste noire de l'AMF.",
		Source: "AMF", SourceURL: "https://amf.example", ObservedAt: time.Date(2026, 11, 2, 9, 14, 42, 0, paris),
	}}, []domain.SourceState{{Source: "AMF", Status: domain.SourceOK}, {Source: "ACPR", Status: domain.SourceUnavailable}})

	want := `{"input":"armen.ltd","type":"domain","checked_at":"2026-11-02T08:25:53Z","summary":null,` +
		`"signals":[{"code":"amf_blacklist","level":"alerte","message":"Figure sur la liste noire de l'AMF.",` +
		`"source":"AMF","source_url":"https://amf.example","observed_at":"2026-11-02T08:14:42Z"}],` +
		`"sources":[{"source":"AMF","status":"ok"},{"source":"ACPR","status":"indisponible"}]}`
	if got := marshal(t, wire.ReportOf(domain.KindDomain, r)); got != want {
		t.Errorf("ReportOf() =\n%s\nwant\n%s", got, want)
	}
}

func TestReportOfWithoutSignalSaysSoAndKeepsEmptyLists(t *testing.T) {
	t.Parallel()

	r := domain.NewReport("552100554", time.Date(2026, 11, 2, 8, 25, 53, 0, time.UTC), nil, nil)

	want := `{"input":"552100554","type":"siren","checked_at":"2026-11-02T08:25:53Z",` +
		`"summary":"Aucun signal connu dans nos sources","signals":[],"sources":[]}`
	if got := marshal(t, wire.ReportOf(domain.KindSIREN, r)); got != want {
		t.Errorf("ReportOf() =\n%s\nwant\n%s", got, want)
	}
}

func TestSourcesOfGivesTheStateOfEachSource(t *testing.T) {
	t.Parallel()

	listed := []app.ListedSource{
		{
			Info:   domain.SourceInfo{Name: "AMF", Publisher: "AMF", Licence: "Licence Ouverte 2.0", URL: "https://amf.example"},
			Synced: true,
			Sync:   app.SourceSync{Source: "AMF", LastSuccess: time.Date(2026, 11, 2, 9, 14, 42, 0, paris), Rows: 3466},
		},
		{
			Info:   domain.SourceInfo{Name: "ARCEP", Publisher: "ARCEP", Licence: "Licence Ouverte 1.0", URL: "https://arcep.example"},
			Synced: true,
		},
		{Info: domain.SourceInfo{Name: "SIRENE", Publisher: "DINUM", Licence: "Licence Ouverte 2.0", URL: "https://sirene.example"}},
	}

	want := `{"sources":[` +
		`{"name":"AMF","publisher":"AMF","licence":"Licence Ouverte 2.0","url":"https://amf.example",` +
		`"status":"ok","updated_at":"2026-11-02T08:14:42Z","entries":3466},` +
		`{"name":"ARCEP","publisher":"ARCEP","licence":"Licence Ouverte 1.0","url":"https://arcep.example",` +
		`"status":"indisponible","updated_at":null,"entries":0},` +
		`{"name":"SIRENE","publisher":"DINUM","licence":"Licence Ouverte 2.0","url":"https://sirene.example",` +
		`"status":"ok","updated_at":null,"entries":null}]}`
	if got := marshal(t, wire.SourcesOf(listed)); got != want {
		t.Errorf("SourcesOf() =\n%s\nwant\n%s", got, want)
	}
}

func TestSourcesOfNoneIsAnEmptyList(t *testing.T) {
	t.Parallel()
	if got, want := marshal(t, wire.SourcesOf(nil)), `{"sources":[]}`; got != want {
		t.Errorf("SourcesOf(nil) = %s, want %s", got, want)
	}
}
