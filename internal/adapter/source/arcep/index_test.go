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

// fixtureIndex returns the index of the fixture snapshot.
func fixtureIndex(tb testing.TB) *Index {
	tb.Helper()
	snap, err := ParseNumbering(bytes.NewReader(readFixture(tb, numberingFixture)), fixtureOperators(tb))
	if err != nil {
		tb.Fatalf("ParseNumbering() error = %v", err)
	}
	ix, err := NewIndex(snap)
	if err != nil {
		tb.Fatalf("NewIndex() error = %v", err)
	}
	return ix
}

func mustPhone(tb testing.TB, s string) domain.Phone {
	tb.Helper()
	p, err := domain.ParsePhone(s)
	if err != nil {
		tb.Fatalf("ParsePhone() error = %v", err)
	}
	return p
}

func TestLookup(t *testing.T) {
	t.Parallel()
	ix := fixtureIndex(t)
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	orange := Assignment{
		Prefix: "0140", Operator: "FRTE", OperatorName: "Orange", OperatorSIREN: "380129866",
		Territory: "Métropole", AssignedOn: day(2017, 1, 1),
	}
	tests := []struct {
		phone string
		want  Assignment
		found bool
	}{
		{"01 40 12 34 56", orange, true},
		{"+33 1 40 00 00 00", orange, true},
		{"01 40 99 99 99", orange, true},
		{"06 12 34 56 78", Assignment{
			Prefix: "0612", Operator: "SFR0", OperatorName: "Société française du radiotéléphone", OperatorSIREN: "343059564",
			Territory: "Métropole", AssignedOn: day(2011, 12, 8),
		}, true},
		{"0690 17 12 34", Assignment{
			Prefix: "069017", Operator: "BUYC", OperatorName: "Digicel Antilles Françaises Guyane", OperatorSIREN: "431416288",
			Territory: "Guadeloupe, St Martin et St Barthélemy", AssignedOn: day(2007, 1, 18),
		}, true},
		{"01 59 14 00 00", Assignment{
			Prefix: "015914", Operator: "VOIC", OperatorName: "8x8 UK Limited",
			Territory: "Métropole", AssignedOn: day(2022, 12, 9),
		}, true},
		{"01 41 00 00 00", Assignment{}, false},
		{"01 39 99 99 99", Assignment{}, false},
		{"+44 20 7946 0958", Assignment{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			t.Parallel()
			got, found := ix.Lookup(mustPhone(t, tt.phone))
			if got != tt.want || found != tt.found {
				t.Errorf("Lookup() = %+v, %v, want %+v, %v", got, found, tt.want, tt.found)
			}
		})
	}
}

func TestLookupZeroPhone(t *testing.T) {
	t.Parallel()
	if _, found := fixtureIndex(t).Lookup(domain.Phone{}); found {
		t.Error("Lookup(zero Phone) found an assignment")
	}
}

func TestNewIndexErrors(t *testing.T) {
	t.Parallel()
	good := []string{"0140", "0140000000", "0140999999", "FRTE", "Orange", "380129866", "Métropole", "2017-01-01"}
	with := func(i int, v string) [][]string {
		row := slices.Clone(good)
		row[i] = v
		return [][]string{row}
	}
	tests := map[string]domain.Snapshot{
		"other source":   {Source: "AMF", Header: numberingColumns, Rows: [][]string{good}},
		"other header":   {Source: NumberingSource, Header: numberingColumns[1:], Rows: [][]string{good[1:]}},
		"short row":      {Source: NumberingSource, Header: numberingColumns, Rows: [][]string{good[:4]}},
		"not a number":   {Source: NumberingSource, Header: numberingColumns, Rows: with(1, "01400000xx")},
		"French date":    {Source: NumberingSource, Header: numberingColumns, Rows: with(7, "01/01/2017")},
		"overlapping":    {Source: NumberingSource, Header: numberingColumns, Rows: [][]string{good, good}},
		"first > last":   {Source: NumberingSource, Header: numberingColumns, Rows: with(1, "0141000000")},
		"lengths differ": {Source: NumberingSource, Header: numberingColumns, Rows: with(2, "014099999")},
	}
	for name, snap := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewIndex(snap); !errors.Is(err, ErrFormat) {
				t.Errorf("NewIndex() error = %v, want %v", err, ErrFormat)
			}
		})
	}
}

func BenchmarkLookup(b *testing.B) {
	ix := fixtureIndex(b)
	p := mustPhone(b, "06 12 34 56 78")
	b.ReportAllocs()
	for b.Loop() {
		if _, found := ix.Lookup(p); !found {
			b.Fatal("not found")
		}
	}
}
