// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// fixturePlan returns the plan of the fixture snapshot.
func fixturePlan(tb testing.TB) *domain.NumberingPlan {
	tb.Helper()
	snap, err := ParseNumbering(bytes.NewReader(readFixture(tb, numberingFixture)), fixtureOperators(tb))
	if err != nil {
		tb.Fatalf("ParseNumbering() error = %v", err)
	}
	plan, err := Plan(snap)
	if err != nil {
		tb.Fatalf("Plan() error = %v", err)
	}
	return plan
}

func mustPhone(tb testing.TB, s string) domain.Phone {
	tb.Helper()
	p, err := domain.ParsePhone(s)
	if err != nil {
		tb.Fatalf("ParsePhone() error = %v", err)
	}
	return p
}

func TestPlanOfTheFixture(t *testing.T) {
	t.Parallel()
	plan := fixturePlan(t)
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	tests := []struct {
		phone string
		want  domain.NumberAssignment
		found bool
	}{
		{"01 40 12 34 56", domain.NumberAssignment{
			Prefix: "0140", Operator: "FRTE", OperatorName: "Orange", OperatorSIREN: "380129866",
			Territory: "Métropole", AssignedOn: day(2017, 1, 1),
		}, true},
		{"0690 17 12 34", domain.NumberAssignment{
			Prefix: "069017", Operator: "BUYC", OperatorName: "Digicel Antilles Françaises Guyane", OperatorSIREN: "431416288",
			Territory: "Guadeloupe, St Martin et St Barthélemy", AssignedOn: day(2007, 1, 18),
		}, true},
		{"01 59 14 00 00", domain.NumberAssignment{
			Prefix: "015914", Operator: "VOIC", OperatorName: "8x8 UK Limited",
			Territory: "Métropole", AssignedOn: day(2022, 12, 9),
		}, true},
		{"01 41 00 00 00", domain.NumberAssignment{}, false},
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

func TestPlanErrors(t *testing.T) {
	t.Parallel()
	good := []string{"0140", "0140000000", "0140999999", "FRTE", "Orange", "380129866", "Métropole", "2017-01-01"}
	with := func(i int, v string) [][]string {
		row := slices.Clone(good)
		row[i] = v
		return [][]string{row}
	}
	tests := map[string]domain.Snapshot{
		"other source": {Source: "AMF", Header: numberingColumns, Rows: [][]string{good}},
		"other header": {Source: NumberingSource, Header: numberingColumns[1:], Rows: [][]string{good[1:]}},
		"short row":    {Source: NumberingSource, Header: numberingColumns, Rows: [][]string{good[:4]}},
		"not a number": {Source: NumberingSource, Header: numberingColumns, Rows: with(1, "01400000xx")},
		"French date":  {Source: NumberingSource, Header: numberingColumns, Rows: with(7, "01/01/2017")},
		"overlapping":  {Source: NumberingSource, Header: numberingColumns, Rows: [][]string{good, good}},
	}
	for name, snap := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Plan(snap); !errors.Is(err, ErrFormat) {
				t.Errorf("Plan() error = %v, want %v", err, ErrFormat)
			}
		})
	}
}
