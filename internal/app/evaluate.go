// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/arnakoff/api/internal/domain"
)

// ErrUnknownLabel means a case is labelled neither scam nor legit.
var ErrUnknownLabel = errors.New("unknown corpus label")

// Label tells what a corpus case is known to be.
type Label string

// Corpus labels.
const (
	// LabelScam is an input an authority blacklisted.
	LabelScam Label = "scam"
	// LabelLegit is an input of a verified legitimate entity.
	LabelLegit Label = "legit"
)

// Case is one labelled input of the evaluation corpus.
type Case struct {
	Input domain.Input
	Label Label
}

// Checker returns the report of an input; *Check is one.
type Checker interface {
	Check(ctx context.Context, in domain.Input) domain.Report
}

// Score counts the corpus cases each signal level reached.
type Score struct {
	// Scams is the number of scam cases.
	Scams int
	// ScamsAlerted counts the scams with at least one alerte.
	ScamsAlerted int
	// ScamsFlagged counts the scams with at least one attention or alerte.
	ScamsFlagged int
	// Legit is the number of legit cases.
	Legit int
	// LegitAlerted counts the legit cases with at least one alerte; it must
	// stay 0.
	LegitAlerted int
	// FalseAlerts lists the legit cases with at least one attention or
	// alerte, in corpus order.
	FalseAlerts []FalseAlert
	// Unavailable counts the cases a source could not be consulted for:
	// their score proves less.
	Unavailable int
}

// FalseAlert is a legit case that got an attention or an alerte.
type FalseAlert struct {
	// Input is the normalised input.
	Input string
	// Level is the most severe level it got.
	Level domain.Level
}

// Evaluate checks every case in turn and scores the reports.
func Evaluate(ctx context.Context, c Checker, cases []Case) (Score, error) {
	var s Score
	for _, k := range cases {
		if err := ctx.Err(); err != nil {
			return Score{}, err
		}
		if err := s.add(k.Label, c.Check(ctx, k.Input)); err != nil {
			return Score{}, err
		}
	}
	return s, nil
}

// add scores the report of a case labelled l.
func (s *Score) add(l Label, r domain.Report) error {
	top := topLevel(r)
	switch l {
	case LabelScam:
		s.Scams++
		s.ScamsAlerted += countIf(top >= domain.LevelAlert)
		s.ScamsFlagged += countIf(top >= domain.LevelAttention)
	case LabelLegit:
		s.Legit++
		s.LegitAlerted += countIf(top >= domain.LevelAlert)
		if top >= domain.LevelAttention {
			s.FalseAlerts = append(s.FalseAlerts, FalseAlert{Input: r.Input(), Level: top})
		}
	default:
		return fmt.Errorf("%w: %q", ErrUnknownLabel, l)
	}
	s.Unavailable += countIf(anyUnavailable(r))
	return nil
}

// topLevel returns the most severe level of r's signals, zero for none.
func topLevel(r domain.Report) domain.Level {
	signals := r.Signals()
	if len(signals) == 0 {
		return 0
	}
	return signals[0].Level // most severe first
}

// anyUnavailable reports whether a source of r could not be consulted.
func anyUnavailable(r domain.Report) bool {
	for _, st := range r.Sources() {
		if st.Status == domain.SourceUnavailable {
			return true
		}
	}
	return false
}

// countIf returns 1 when b holds, 0 otherwise.
func countIf(b bool) int {
	if b {
		return 1
	}
	return 0
}
