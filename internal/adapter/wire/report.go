// SPDX-License-Identifier: AGPL-3.0-only

package wire

import (
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Report is a report as the contract's Report schema describes it.
type Report struct {
	Input     string        `json:"input"`
	Type      string        `json:"type"`
	CheckedAt string        `json:"checked_at"`
	Summary   *string       `json:"summary"`
	Signals   []Signal      `json:"signals"`
	Sources   []SourceState `json:"sources"`
}

// Signal is a signal as the contract's Signal schema describes it.
type Signal struct {
	Code       string `json:"code"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	Advice     string `json:"advice,omitempty"`
	Source     string `json:"source"`
	SourceURL  string `json:"source_url"`
	ObservedAt string `json:"observed_at"`
}

// SourceState is a source state as the contract's SourceState schema
// describes it.
type SourceState struct {
	Source string `json:"source"`
	Status string `json:"status"`
}

// noSignalSummary is the summary of a report without any signal; the
// reports share it rather than allocate a copy each.
var noSignalSummary = domain.NoSignalSummary

// ReportOf returns the representation of r, the report of an input of
// kind k.
func ReportOf(k domain.Kind, r domain.Report) Report {
	signals, sources := r.Signals(), r.Sources()
	body := Report{
		Input:     r.Input(),
		Type:      k.String(),
		CheckedAt: timestamp(r.CheckedAt()),
		Signals:   make([]Signal, 0, len(signals)),
		Sources:   make([]SourceState, 0, len(sources)),
	}
	if r.Summary() != "" {
		body.Summary = &noSignalSummary
	}
	for _, s := range signals {
		body.Signals = append(body.Signals, Signal{
			Code: s.Code, Level: s.Level.String(), Message: s.Message, Advice: s.Advice,
			Source: string(s.Source), SourceURL: s.SourceURL, ObservedAt: timestamp(s.ObservedAt),
		})
	}
	for _, s := range sources {
		body.Sources = append(body.Sources, SourceState{Source: string(s.Source), Status: s.Status.String()})
	}
	return body
}

// timestamp writes t in RFC 3339, in UTC, to the second.
func timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
