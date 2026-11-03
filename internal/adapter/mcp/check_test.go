// SPDX-License-Identifier: AGPL-3.0-only

package mcp_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/mcp"
	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/domain"
)

// blacklisted is a report with a signal and every source status, with
// times in another zone than UTC.
func blacklisted() domain.Report {
	return domain.NewReport("armen.ltd", time.Date(2026, 11, 2, 9, 25, 53, 0, paris), []domain.Signal{{
		Code:       "amf_blacklist",
		Level:      domain.LevelAlert,
		Message:    "Figure sur la liste noire de l'AMF (catégorie Usurpation) depuis le 08/10/2026.",
		Source:     "AMF",
		SourceURL:  "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde",
		ObservedAt: time.Date(2026, 11, 2, 9, 14, 42, 0, paris),
	}}, []domain.SourceState{
		{Source: "AMF", Status: domain.SourceOK},
		{Source: "ACPR", Status: domain.SourceUnavailable},
		{Source: "ARCEP", Status: domain.SourceNotApplicable},
	})
}

func TestCheckAnswersTheReportOfTheDetectedInput(t *testing.T) {
	t.Parallel()

	c := &checker{report: blacklisted()}
	cs := connect(t, mcp.Options{Check: c, Sources: sourceLister{}})

	res := call(t, cs, "check", map[string]any{"input": "https://armen.ltd/offre"})

	if res.IsError {
		t.Fatalf("check is an error: %s", text(t, res))
	}
	if got := c.checked(); len(got) != 1 || got[0].Kind() != domain.KindDomain || got[0].String() != "armen.ltd" {
		t.Errorf("checked %v, want the domain armen.ltd", got)
	}
	holdToContract(t, res, "Report")
	sameJSON(t, "report", res.StructuredContent, wire.ReportOf(domain.KindDomain, blacklisted()))
}

func TestCheckForcesTheTypeItIsGiven(t *testing.T) {
	t.Parallel()

	c := &checker{report: domain.NewReport("552100554", time.Date(2026, 11, 2, 8, 25, 53, 0, time.UTC), nil, nil)}
	cs := connect(t, mcp.Options{Check: c, Sources: sourceLister{}})

	res := call(t, cs, "check", map[string]any{"input": "552 100 554", "type": "siren"})

	if res.IsError {
		t.Fatalf("check is an error: %s", text(t, res))
	}
	if got := c.checked(); len(got) != 1 || got[0].Kind() != domain.KindSIREN {
		t.Errorf("checked %v, want a SIREN", got)
	}
	holdToContract(t, res, "Report")
	if got := res.StructuredContent.(map[string]any)["summary"]; got != domain.NoSignalSummary {
		t.Errorf("summary = %v, want %q", got, domain.NoSignalSummary)
	}
}

func TestCheckRefusesAnInvalidInputWithTheAPIWording(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		args map[string]any
		want string
	}{
		{"detected", map[string]any{"input": "pas une entrée"}, wire.InvalidInput(domain.KindAuto)},
		{"empty", map[string]any{"input": ""}, wire.InvalidInput(domain.KindAuto)},
		{"forced", map[string]any{"input": "armen.ltd", "type": "iban"}, wire.InvalidInput(domain.KindIBAN)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &checker{}
			cs := connect(t, mcp.Options{Check: c, Sources: sourceLister{}})

			res := call(t, cs, "check", tt.args)

			if !res.IsError {
				t.Fatal("check is not an error")
			}
			if got := text(t, res); got != tt.want {
				t.Errorf("error = %q, want %q", got, tt.want)
			}
			if got := c.checked(); len(got) != 0 {
				t.Errorf("checked %v, want nothing", got)
			}
		})
	}
}

func TestCheckRefusesArgumentsOutsideItsSchema(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		args map[string]any
	}{
		{"no input", map[string]any{}},
		{"unknown type", map[string]any{"input": "armen.ltd", "type": "auto"}},
		{"unknown argument", map[string]any{"input": "armen.ltd", "url": "https://armen.ltd"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &checker{}
			cs := connect(t, mcp.Options{Check: c, Sources: sourceLister{}})

			if res := call(t, cs, "check", tt.args); !res.IsError {
				t.Error("check is not an error")
			}
			if got := c.checked(); len(got) != 0 {
				t.Errorf("checked %v, want nothing", got)
			}
		})
	}
}
