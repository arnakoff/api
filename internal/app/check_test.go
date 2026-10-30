// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestCheckConsultsTheRegistriesOfTheInputKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		domains []string // inputs the domain registry was asked about
		sirens  []string // inputs the company and notice registries were asked about
		want    []domain.SourceState
	}{
		{"domain", "www.exemple.fr", []string{"www.exemple.fr"}, nil, states(okState, naState, naState)},
		{"email", "contact@exemple.fr", []string{"exemple.fr"}, nil, states(okState, naState, naState)},
		{"SIREN", "120027016", nil, []string{"120027016"}, states(naState, okState, okState)},
		{"SIRET", "12002701600563", nil, []string{"120027016"}, states(naState, okState, okState)},
		{"phone", "+33612345678", nil, nil, states(naState, naState, naState)},
		{"IBAN", "FR1420041010050500013M02606", nil, nil, states(naState, naState, naState)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newRegistries()
			got := app.NewCheck(r.all()).Check(context.Background(), parse(t, tt.in))

			if got.Input() != parse(t, tt.in).String() {
				t.Errorf("Input() = %q, want the normalised input", got.Input())
			}
			if !slices.Equal(got.Sources(), tt.want) {
				t.Errorf("Sources() = %v, want %v", got.Sources(), tt.want)
			}
			if len(got.Signals()) != 0 {
				t.Errorf("Signals() = %v, want none: no rule is wired yet", got.Signals())
			}
			r.assertAsked(t, tt.domains, tt.sirens)
		})
	}
}

func TestCheckStatusFollowsTheRegistryAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want domain.SourceStatus
	}{
		{"answered", nil, domain.SourceOK},
		// An answer that the registry holds no such entry is an answer.
		{"not found", fmt.Errorf("rdap: %w", app.ErrNotFound), domain.SourceOK},
		{"not covered", fmt.Errorf("rdap: %w", app.ErrNotCovered), domain.SourceNotApplicable},
		{"failed", errors.New("connection reset"), domain.SourceUnavailable},
		{"cancelled", context.Canceled, domain.SourceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newRegistries()
			r.domains.err, r.companies.err, r.notices.err = tt.err, tt.err, tt.err
			check := app.NewCheck(r.all())

			for _, in := range []string{"exemple.fr", "120027016"} {
				for _, s := range check.Check(context.Background(), parse(t, in)).Sources() {
					if s.Status != domain.SourceNotApplicable && s.Status != tt.want {
						t.Errorf("%s: %s status = %v, want %v", in, s.Source, s.Status, tt.want)
					}
				}
			}
		})
	}
}

func TestCheckLeavesOutAMissingRegistry(t *testing.T) {
	t.Parallel()

	r := newRegistries()
	regs := r.all()
	regs.Domains = nil

	got := app.NewCheck(regs).Check(context.Background(), parse(t, "exemple.fr")).Sources()

	want := []domain.SourceState{{Source: "SIRENE", Status: naState}, {Source: "BODACC", Status: naState}}
	if !slices.Equal(got, want) {
		t.Errorf("Sources() = %v, want %v", got, want)
	}
}

// Shorthands for the source statuses.
const (
	okState = domain.SourceOK
	naState = domain.SourceNotApplicable
)

// states returns the source states of the fake RDAP, SIRENE and BODACC
// registries, in that order.
func states(rdap, sirene, bodacc domain.SourceStatus) []domain.SourceState {
	return []domain.SourceState{
		{Source: "RDAP", Status: rdap},
		{Source: "SIRENE", Status: sirene},
		{Source: "BODACC", Status: bodacc},
	}
}

// parse parses in, detecting its kind.
func parse(t *testing.T, in string) domain.Input {
	t.Helper()
	v, err := domain.ParseInput(in, domain.KindAuto)
	if err != nil {
		t.Fatalf("ParseInput(%q): %v", in, err)
	}
	return v
}

func TestRegistriesWithout(t *testing.T) {
	t.Parallel()

	r := newRegistries().all()
	got := r.Without(func(s domain.Source) bool { return s == "RDAP" || s == "BODACC" })

	if got.Domains != nil || got.Companies == nil || got.Notices != nil {
		t.Errorf("Without(RDAP, BODACC) = %+v, want SIRENE only", got)
	}
	if none := (app.Registries{}).Without(func(domain.Source) bool { return true }); none != (app.Registries{}) {
		t.Errorf("Without on no registry = %+v, want none", none)
	}
}
