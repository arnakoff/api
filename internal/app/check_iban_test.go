// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestCheckStatesTheCountryOfAnIBAN(t *testing.T) {
	t.Parallel()

	r := newRegistries()
	report := app.NewCheck(r.all(), &clock{now: t0}).Check(context.Background(), parse(t, "LT12 1000 0111 0100 1000"))

	want := domain.Signal{
		Code:       "iban_country",
		Level:      domain.LevelInfo,
		Message:    "Pays de tenue du compte : Lituanie (code LT de l'IBAN).",
		Source:     "IBAN",
		SourceURL:  "https://www.swift.com/standards/data-standards/iban-international-bank-account-number",
		ObservedAt: t0,
	}
	if got := report.Signals(); len(got) != 1 || got[0] != want {
		t.Errorf("Signals() = %+v, want [%+v]", got, want)
	}
	// The IBAN is read, not consulted: it is no source of the report.
	if got, want := report.Sources(), states(naState, naState, naState); !slices.Equal(got, want) {
		t.Errorf("Sources() = %v, want %v", got, want)
	}
}

func TestCheckStatesNoIBANCountryForOtherInputs(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"exemple.fr", "contact@exemple.fr", "+33612345678", "120027016"} {
		r := newRegistries()
		for _, s := range app.NewCheck(r.all(), &clock{now: t0}).Check(context.Background(), parse(t, in)).Signals() {
			if s.Code == "iban_country" {
				t.Errorf("%s: Signals() holds %+v", in, s)
			}
		}
	}
}
