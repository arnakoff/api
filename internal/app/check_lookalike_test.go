// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// ameliProof is the proof page of ameli.fr in the reference list.
const ameliProof = "https://www.ameli.fr/assure/mentions-legales-cgu"

func officials(t *testing.T) signal.Officials {
	t.Helper()
	o, err := signal.NewOfficials([]signal.Official{{Domain: "ameli.fr", Names: []string{"ameli"}, Of: "de l'Assurance Maladie", Proof: ameliProof}})
	if err != nil {
		t.Fatalf("NewOfficials: %v", err)
	}
	return o
}

func TestCheckFlagsALookalikeDomain(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"ameli-remboursement.com", "contact@ameli-remboursement.com"} {
		regs := newRegistries().all()
		regs.Officials = officials(t)
		report := app.NewCheck(regs, &clock{now: t0}).Check(context.Background(), parse(t, in))

		want := domain.Signal{
			Code:       "domain_lookalike",
			Level:      domain.LevelAttention,
			Message:    "Le domaine ameli-remboursement.com ressemble à ameli.fr, site officiel de l'Assurance Maladie, mais n'en fait pas partie.",
			Advice:     "Tape toi-même l'adresse officielle plutôt que de suivre un lien reçu.",
			Source:     "Domaines officiels",
			SourceURL:  ameliProof,
			ObservedAt: t0,
		}
		if got := report.Signals(); !slices.Contains(got, want) {
			t.Errorf("%s: Signals() = %+v, want it to hold %+v", in, got, want)
		}
	}
}

func TestCheckFlagsNoLookalikeWithoutOfficials(t *testing.T) {
	t.Parallel()

	r := newRegistries()
	for _, s := range app.NewCheck(r.all(), &clock{now: t0}).Check(context.Background(), parse(t, "ameli-remboursement.com")).Signals() {
		if s.Code == "domain_lookalike" {
			t.Errorf("Signals() holds %+v", s)
		}
	}
}

func TestRegistriesWithoutLeavesOutTheOfficials(t *testing.T) {
	t.Parallel()

	regs := newRegistries().all()
	regs.Officials = officials(t)
	regs = regs.Without(func(s domain.Source) bool { return s == signal.LookalikeSource })
	for _, s := range app.NewCheck(regs, &clock{now: t0}).Check(context.Background(), parse(t, "ameli-remboursement.com")).Signals() {
		if s.Code == "domain_lookalike" {
			t.Errorf("Signals() holds %+v", s)
		}
	}
}
