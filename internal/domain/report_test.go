// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

var observed = time.Date(2026, time.October, 29, 9, 0, 0, 0, time.UTC)

func signal(code string, level domain.Level) domain.Signal {
	return domain.Signal{
		Code:       code,
		Level:      level,
		Message:    "message " + code,
		Source:     domain.Source("source " + code),
		SourceURL:  "https://example.org/" + code,
		ObservedAt: observed,
	}
}

func codes(signals []domain.Signal) []string {
	out := make([]string, 0, len(signals))
	for _, s := range signals {
		out = append(out, s.Code)
	}
	return out
}

func TestNewReport_sortsSignalsMostSevereFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		signals []domain.Signal
		want    []string
	}{
		{"no signal", nil, []string{}},
		{
			"mixed levels",
			[]domain.Signal{
				signal("a", domain.LevelInfo),
				signal("b", domain.LevelAlert),
				signal("c", domain.LevelAttention),
			},
			[]string{"b", "c", "a"},
		},
		{
			"equal levels keep their order",
			[]domain.Signal{
				signal("a", domain.LevelAttention),
				signal("b", domain.LevelAlert),
				signal("c", domain.LevelAttention),
				signal("d", domain.LevelAlert),
			},
			[]string{"b", "d", "a", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := domain.NewReport("example.fr", tt.signals, nil)
			if got := codes(r.Signals()); !slices.Equal(got, tt.want) {
				t.Errorf("Signals() codes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewReport_keepsInputAndSources(t *testing.T) {
	t.Parallel()

	sources := []domain.SourceState{
		{Source: "AMF", Status: domain.SourceOK},
		{Source: "RDAP", Status: domain.SourceUnavailable},
		{Source: "SIRENE", Status: domain.SourceNotApplicable},
	}
	r := domain.NewReport("example.fr", nil, sources)

	if got := r.Input(); got != "example.fr" {
		t.Errorf("Input() = %q, want %q", got, "example.fr")
	}
	if got := r.Sources(); !slices.Equal(got, sources) {
		t.Errorf("Sources() = %v, want %v", got, sources)
	}
}

func TestNewReport_isNotChangedThroughSlices(t *testing.T) {
	t.Parallel()

	signals := []domain.Signal{signal("a", domain.LevelInfo), signal("b", domain.LevelAlert)}
	sources := []domain.SourceState{{Source: "AMF", Status: domain.SourceOK}}
	r := domain.NewReport("example.fr", signals, sources)

	if got := codes(signals); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("caller's signals reordered to %v", got)
	}
	signals[0].Code = "changed"
	sources[0].Status = domain.SourceUnavailable
	r.Signals()[0].Code = "changed"
	r.Sources()[0].Status = domain.SourceUnavailable

	if got := codes(r.Signals()); !slices.Equal(got, []string{"b", "a"}) {
		t.Errorf("Signals() codes = %v after outside writes, want [b a]", got)
	}
	if got := r.Sources()[0].Status; got != domain.SourceOK {
		t.Errorf("Sources()[0].Status = %v after outside writes, want ok", got)
	}
}
