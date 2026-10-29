// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"cmp"
	"slices"
)

// Report is the result of checking one input: its signals, most severe
// first, and the state of each source consulted. A Report is immutable: its
// accessors return copies.
type Report struct {
	input   string
	signals []Signal
	sources []SourceState
}

// NewReport builds the report for the normalised input. Signals are sorted
// by level, most severe first; signals of equal level keep their order. The
// slices are copied, the caller's are left untouched.
func NewReport(input string, signals []Signal, sources []SourceState) Report {
	sorted := slices.Clone(signals)
	slices.SortStableFunc(sorted, func(a, b Signal) int {
		return cmp.Compare(b.Level, a.Level)
	})
	return Report{input: input, signals: sorted, sources: slices.Clone(sources)}
}

// Input returns the normalised input that was checked.
func (r Report) Input() string { return r.input }

// Signals returns a copy of the signals, most severe first.
func (r Report) Signals() []Signal { return slices.Clone(r.signals) }

// Sources returns a copy of the state of each source.
func (r Report) Sources() []SourceState { return slices.Clone(r.sources) }
