// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// Ranges of the ARCEP numbering fixture (testdata/arcep), checked on
// 29/10/2026.
var (
	orange = domain.NumberAssignment{
		Prefix: "0140", Operator: "FRTE", OperatorName: "Orange", OperatorSIREN: "380129866",
		Territory: "Métropole", AssignedOn: day(2017, 1, 1),
	}
	sfr = domain.NumberAssignment{
		Prefix: "0612", Operator: "SFR0", OperatorName: "Société française du radiotéléphone", OperatorSIREN: "343059564",
		Territory: "Métropole", AssignedOn: day(2011, 12, 8),
	}
	digicel = domain.NumberAssignment{
		Prefix: "069017", Operator: "BUYC", OperatorName: "Digicel Antilles Françaises Guyane", OperatorSIREN: "431416288",
		Territory: "Guadeloupe, St Martin et St Barthélemy", AssignedOn: day(2007, 1, 18),
	}
	short = domain.NumberAssignment{Prefix: "3620", Operator: "FRTE", Territory: "National", AssignedOn: day(2000, 1, 1)}
)

func mustRange(tb testing.TB, first, last string, a domain.NumberAssignment) domain.NumberRange {
	tb.Helper()
	r, err := domain.NewNumberRange(first, last, a)
	if err != nil {
		tb.Fatalf("NewNumberRange(%q, %q) error = %v", first, last, err)
	}
	return r
}

func testPlan(tb testing.TB) *domain.NumberingPlan {
	tb.Helper()
	plan, err := domain.NewNumberingPlan([]domain.NumberRange{
		mustRange(tb, "0612000000", "0612999999", sfr),
		mustRange(tb, "0140000000", "0140999999", orange),
		mustRange(tb, "3620", "3620", short),
		mustRange(tb, "0690170000", "0690179999", digicel),
	})
	if err != nil {
		tb.Fatalf("NewNumberingPlan() error = %v", err)
	}
	return plan
}

func TestNumberingPlanLookup(t *testing.T) {
	t.Parallel()
	plan := testPlan(t)
	tests := []struct {
		phone string
		want  domain.NumberAssignment
		found bool
	}{
		{"01 40 12 34 56", orange, true},
		{"+33 1 40 00 00 00", orange, true},
		{"01 40 99 99 99", orange, true},
		{"06 12 34 56 78", sfr, true},
		{"0690 17 12 34", digicel, true},
		{"+590 690 17 99 99", digicel, true},
		{"01 41 00 00 00", domain.NumberAssignment{}, false},
		{"01 39 99 99 99", domain.NumberAssignment{}, false},
		{"07 00 00 00 00", domain.NumberAssignment{}, false},
		{"+44 20 7946 0958", domain.NumberAssignment{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			t.Parallel()
			got, found := plan.Lookup(mustPhone(t, tt.phone))
			if got != tt.want || found != tt.found {
				t.Errorf("Lookup() = %+v, %v, want %+v, %v", got, found, tt.want, tt.found)
			}
		})
	}
}

func TestNumberingPlanLookupZeroPhone(t *testing.T) {
	t.Parallel()
	if _, found := testPlan(t).Lookup(domain.Phone{}); found {
		t.Error("Lookup(zero Phone) found an assignment")
	}
}

func TestNumberingPlanCovers(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"06 12 34 56 78":   true,
		"01 41 00 00 00":   true,
		"0690 17 12 34":    true,
		"+44 20 7946 0958": false,
	}
	for in, want := range tests {
		if got := domain.NumberingPlanCovers(mustPhone(t, in)); got != want {
			t.Errorf("NumberingPlanCovers(%q) = %v, want %v", in, got, want)
		}
	}
	if domain.NumberingPlanCovers(domain.Phone{}) {
		t.Error("NumberingPlanCovers(zero Phone) = true")
	}
}

func TestNewNumberRangeErrors(t *testing.T) {
	t.Parallel()
	tests := map[string][2]string{
		"empty":                 {"", ""},
		"not a number":          {"01400000xx", "0140999999"},
		"sign":                  {"+140000000", "0140999999"},
		"lengths differ":        {"0140000000", "014099999"},
		"first after last":      {"0141000000", "0140999999"},
		"national too short":    {"014000000", "014099999"},
		"short number too long": {"1400000", "1400099"},
		"overflows":             {"090000000000000000000", "090099999999999999999"},
	}
	for name, b := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := domain.NewNumberRange(b[0], b[1], orange); !errors.Is(err, domain.ErrInvalidRange) {
				t.Errorf("NewNumberRange(%q, %q) error = %v, want %v", b[0], b[1], err, domain.ErrInvalidRange)
			}
		})
	}
}

func TestNewNumberingPlanRejectsOverlaps(t *testing.T) {
	t.Parallel()
	tests := map[string][]domain.NumberRange{
		"same range":   {mustRange(t, "0140000000", "0140999999", orange), mustRange(t, "0140000000", "0140999999", orange)},
		"inside":       {mustRange(t, "0140000000", "0140999999", orange), mustRange(t, "0140500000", "0140500099", sfr)},
		"shared bound": {mustRange(t, "0140000000", "0140999999", orange), mustRange(t, "0140999999", "0141000099", sfr)},
	}
	for name, ranges := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := domain.NewNumberingPlan(ranges); !errors.Is(err, domain.ErrInvalidRange) {
				t.Errorf("NewNumberingPlan() error = %v, want %v", err, domain.ErrInvalidRange)
			}
		})
	}
}

func TestNewNumberingPlanKeepsTheCallersRanges(t *testing.T) {
	t.Parallel()
	ranges := []domain.NumberRange{
		mustRange(t, "0612000000", "0612999999", sfr),
		mustRange(t, "0140000000", "0140999999", orange),
	}
	if _, err := domain.NewNumberingPlan(ranges); err != nil {
		t.Fatal(err)
	}
	if ranges[0] != mustRange(t, "0612000000", "0612999999", sfr) {
		t.Error("NewNumberingPlan() reordered the caller's slice")
	}
}

func BenchmarkNumberingPlanLookup(b *testing.B) {
	plan := testPlan(b)
	p := mustPhone(b, "06 12 34 56 78")
	b.ReportAllocs()
	for b.Loop() {
		if _, found := plan.Lookup(p); !found {
			b.Fatal("not found")
		}
	}
}
