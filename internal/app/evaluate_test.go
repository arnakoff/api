// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestEvaluateCountsTheCasesEachLevelReaches(t *testing.T) {
	t.Parallel()

	checker := fakeChecker{
		"alerted.fr":   {signals: []domain.Level{domain.LevelInfo, domain.LevelAlert}},
		"flagged.fr":   {signals: []domain.Level{domain.LevelAttention}},
		"informed.fr":  {signals: []domain.Level{domain.LevelInfo}},
		"bank.fr":      {signals: []domain.Level{domain.LevelAttention}},
		"ministry.fr":  {signals: []domain.Level{domain.LevelAlert}},
		"unreached.fr": {unavailable: true},
	}
	cases := []app.Case{
		scamCase(t, "alerted.fr"),
		scamCase(t, "flagged.fr"),
		scamCase(t, "informed.fr"),
		scamCase(t, "unreached.fr"),
		legitCase(t, "bank.fr"),
		legitCase(t, "ministry.fr"),
		legitCase(t, "quiet.fr"),
	}

	got, err := app.Evaluate(context.Background(), checker, cases)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	want := app.Score{
		Scams:        4,
		ScamsAlerted: 1,
		// An alerte is at least an attention.
		ScamsFlagged: 2,
		Legit:        3,
		LegitAlerted: 1,
		FalseAlerts: []app.FalseAlert{
			{Input: "bank.fr", Level: domain.LevelAttention},
			{Input: "ministry.fr", Level: domain.LevelAlert},
		},
		Unavailable: 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Evaluate = %+v\nwant %+v", got, want)
	}
}

func TestEvaluateStopsWithTheContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := app.Evaluate(ctx, fakeChecker{}, []app.Case{scamCase(t, "exemple.fr")})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Evaluate error = %v, want context.Canceled", err)
	}
}

func TestEvaluateRejectsAnUnknownLabel(t *testing.T) {
	t.Parallel()

	c := scamCase(t, "exemple.fr")
	c.Label = "suspect"

	_, err := app.Evaluate(context.Background(), fakeChecker{}, []app.Case{c})
	if !errors.Is(err, app.ErrUnknownLabel) {
		t.Errorf("Evaluate error = %v, want ErrUnknownLabel", err)
	}
}

// answer is what the fake checker reports about an input.
type answer struct {
	signals     []domain.Level
	unavailable bool
}

// fakeChecker reports the answer it holds for each input, nothing for the
// others.
type fakeChecker map[string]answer

func (f fakeChecker) Check(_ context.Context, in domain.Input) domain.Report {
	a := f[in.String()]
	signals := make([]domain.Signal, len(a.signals))
	for i, l := range a.signals {
		signals[i] = domain.Signal{Code: "fake_" + l.String(), Level: l}
	}
	states := []domain.SourceState{{Source: "RDAP", Status: domain.SourceOK}}
	if a.unavailable {
		states = append(states, domain.SourceState{Source: "SIRENE", Status: domain.SourceUnavailable})
	}
	return domain.NewReport(in.String(), signals, states)
}

func scamCase(t *testing.T, in string) app.Case {
	t.Helper()
	return app.Case{Input: parse(t, in), Label: app.LabelScam}
}

func legitCase(t *testing.T, in string) app.Case {
	t.Helper()
	return app.Case{Input: parse(t, in), Label: app.LabelLegit}
}
