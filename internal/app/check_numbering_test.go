// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// fakeNumbering serves plan, fetched at fetchedAt, or err.
type fakeNumbering struct {
	plan *domain.NumberingPlan
	err  error
}

func (f fakeNumbering) Numbering() (*domain.NumberingPlan, time.Time, error) {
	if f.err != nil {
		return nil, time.Time{}, f.err
	}
	return f.plan, fetchedAt, nil
}

// sfrPlan holds the range 0612 of the ARCEP numbering fixture.
func sfrPlan(t testing.TB) *domain.NumberingPlan {
	t.Helper()
	r, err := domain.NewNumberRange("0612000000", "0612999999", domain.NumberAssignment{
		Prefix: "0612", Operator: "SFR0", OperatorName: "Société française du radiotéléphone",
		Territory: "Métropole", AssignedOn: time.Date(2011, time.December, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewNumberingPlan([]domain.NumberRange{r})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestCheckReadsTheNumberingPlan(t *testing.T) {
	t.Parallel()

	check := app.NewCheck(app.Registries{Numbering: []app.Numbering{{Reader: fakeNumbering{plan: sfrPlan(t)}}}}, &clock{now: t0})
	tests := []struct {
		in    string
		codes []string
		state domain.SourceStatus
	}{
		{"06 12 34 56 78", []string{"phone_operator"}, okState},
		{"01 41 00 00 00", []string{"phone_unassigned"}, okState},
		{"+44 20 7946 0958", nil, naState},
		{"exemple.fr", nil, naState},
		{"120027016", nil, naState},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			got := check.Check(context.Background(), parse(t, tt.in))

			if codes := signalCodes(got.Signals()); !slices.Equal(codes, tt.codes) {
				t.Errorf("signal codes = %v, want %v", codes, tt.codes)
			}
			want := []domain.SourceState{{Source: "ARCEP", Status: tt.state}}
			if !slices.Equal(got.Sources(), want) {
				t.Errorf("Sources() = %v, want %v", got.Sources(), want)
			}
		})
	}
}

func TestCheckDatesANumberingSignalWithThePlansFetch(t *testing.T) {
	t.Parallel()

	check := app.NewCheck(app.Registries{Numbering: []app.Numbering{{Reader: fakeNumbering{plan: sfrPlan(t)}}}}, &clock{now: t0})

	got := check.Check(context.Background(), parse(t, "06 12 34 56 78")).Signals()

	if len(got) != 1 || !got[0].ObservedAt.Equal(fetchedAt) {
		t.Errorf("Signals() = %v, want one observed at %v", got, fetchedAt)
	}
}

func TestCheckReportsANumberingPlanWithoutSnapshotUnavailable(t *testing.T) {
	t.Parallel()

	regs := app.Registries{Numbering: []app.Numbering{{Reader: fakeNumbering{err: fmt.Errorf("arcep: %w", app.ErrNoSnapshot)}}}}

	got := app.NewCheck(regs, &clock{now: t0}).Check(context.Background(), parse(t, "01 41 00 00 00"))

	if len(got.Signals()) != 0 {
		t.Errorf("Signals() = %v, want none", got.Signals())
	}
	want := []domain.SourceState{{Source: "ARCEP", Status: domain.SourceUnavailable}}
	if !slices.Equal(got.Sources(), want) {
		t.Errorf("Sources() = %v, want %v", got.Sources(), want)
	}
}

func TestCheckListsTheNumberingPlanAfterTheWhitelists(t *testing.T) {
	t.Parallel()

	regs := newRegistries().all()
	regs.Whitelists = whitelists(fakeWhitelist{list: domain.NewWhitelist()}, fakeWhitelist{list: domain.NewWhitelist()})
	regs.Numbering = []app.Numbering{{Reader: fakeNumbering{plan: sfrPlan(t)}}}

	got := app.NewCheck(regs, &clock{now: t0}).Check(context.Background(), parse(t, "06 12 34 56 78")).Sources()

	want := append([]domain.SourceState{
		{Source: "AMF PSAN", Status: okState}, {Source: "AMF biens divers", Status: okState}, {Source: "ARCEP", Status: okState},
	}, states(naState, naState, naState)...)
	if !slices.Equal(got, want) {
		t.Errorf("Sources() = %v, want %v", got, want)
	}
}

func TestRegistriesWithoutTheNumberingPlan(t *testing.T) {
	t.Parallel()

	regs := app.Registries{Numbering: []app.Numbering{{Reader: fakeNumbering{}}}}

	if got := regs.Without(func(s domain.Source) bool { return s == signal.NumberingSource }); len(got.Numbering) != 0 {
		t.Errorf("Without(ARCEP) numbering = %v, want none", got.Numbering)
	}
	if got := regs.Without(func(domain.Source) bool { return false }); len(got.Numbering) != 1 {
		t.Error("Without(nothing) dropped the numbering plan")
	}
	if len(regs.Numbering) != 1 {
		t.Error("Without changed the registries it was called on")
	}
}

func BenchmarkCheckWithTheNumberingPlan(b *testing.B) {
	regs := benchRegistries{}.all()
	regs.Numbering = []app.Numbering{{Reader: fakeNumbering{plan: sfrPlan(b)}}}
	check := app.NewCheck(regs, &clock{now: t0})
	for _, in := range []string{"+33612345678", "+33141000000"} {
		v := parse(b, in)
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			ctx := context.Background()
			for b.Loop() {
				check.Check(ctx, v)
			}
		})
	}
}
