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

// NewReport builds the report for the normalised input. A signal repeating
// the code, source and message of an earlier one is dropped. Signals are
// sorted by level, most severe first; signals of equal level keep their
// order. The slices are copied, the caller's are left untouched.
func NewReport(input string, signals []Signal, sources []SourceState) Report {
	sorted := distinct(signals)
	slices.SortStableFunc(sorted, func(a, b Signal) int {
		return cmp.Compare(b.Level, a.Level)
	})
	return Report{input: input, signals: sorted, sources: slices.Clone(sources)}
}

// distinct returns a copy of signals without the ones repeating the code,
// source and message of an earlier one.
func distinct(signals []Signal) []Signal {
	type fact struct {
		code, message string
		source        Source
	}
	seen := make(map[fact]struct{}, len(signals))
	out := make([]Signal, 0, len(signals))
	for _, s := range signals {
		k := fact{s.Code, s.Message, s.Source}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}
	return out
}

// Input returns the normalised input that was checked.
func (r Report) Input() string { return r.input }

// Signals returns a copy of the signals, most severe first.
func (r Report) Signals() []Signal { return slices.Clone(r.signals) }

// Sources returns a copy of the state of each source.
func (r Report) Sources() []SourceState { return slices.Clone(r.sources) }

// NoSignalSummary is the summary of a report without any signal. It never
// says the input is safe: an absence from the sources proves nothing.
const NoSignalSummary = "Aucun signal connu dans nos sources"

// Summary returns NoSignalSummary when the report has no signal, and ""
// otherwise: the signals speak for themselves.
func (r Report) Summary() string {
	if len(r.signals) == 0 {
		return NoSignalSummary
	}
	return ""
}
