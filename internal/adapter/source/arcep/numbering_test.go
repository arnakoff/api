// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// numberingHeaderLine is the header of the published numbering file, in
// Windows-1252.
const numberingHeaderLine = "EZABPQM;Tranche_Debut;Tranche_Fin;Mn\xe9mo;Territoire;Date_Attribution\r\n"

// fixtureOperators returns the operators of the fixture.
func fixtureOperators(tb testing.TB) map[string]Operator {
	tb.Helper()
	ops, err := ParseOperators(bytes.NewReader(readFixture(tb, operatorsFixture)))
	if err != nil {
		tb.Fatalf("ParseOperators() error = %v", err)
	}
	return ops
}

func TestParseNumberingFixture(t *testing.T) {
	t.Parallel()
	snap, err := ParseNumbering(bytes.NewReader(readFixture(t, numberingFixture)), fixtureOperators(t))
	if err != nil {
		t.Fatalf("ParseNumbering() error = %v", err)
	}
	if snap.Source != NumberingSource || !slices.Equal(snap.Header, numberingColumns) {
		t.Errorf("ParseNumbering() = %s %q, want %s %q", snap.Source, snap.Header, NumberingSource, numberingColumns)
	}
	if len(snap.Rows) != 34 {
		t.Errorf("len(Rows) = %d, want 34", len(snap.Rows))
	}
	if want := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC); !snap.Newest.Equal(want) {
		t.Errorf("Newest = %v, want %v", snap.Newest, want)
	}
	want := [][]string{
		{"0140", "0140000000", "0140999999", "FRTE", "Orange", "380129866", "Métropole", "2017-01-01"},
		{"015914", "0159140000", "0159149999", "VOIC", "8x8 UK Limited", "", "Métropole", "2022-12-09"},
		{"059000", "0590000000", "0590009999", "STMA", "Saint Martin cable", "351955216", "Guadeloupe, St Martin et St Barthélemy", "2014-07-03"},
		{"0700000", "07000000000000", "07000009999999", "FRTE", "Orange", "380129866", "Métropole", "2017-01-01"},
		{"3996", "3996", "3996", "JAG", "Free Pro", "439099656", "National", "2023-04-27"},
	}
	for _, w := range want {
		if !slices.ContainsFunc(snap.Rows, func(row []string) bool { return slices.Equal(row, w) }) {
			t.Errorf("row %q not found", w)
		}
	}
}

func TestParseNumberingUnknownOperator(t *testing.T) {
	t.Parallel()
	// The operators file is updated weekly, the numbering after each board
	// meeting: a new holder may be missing for a few days.
	in := numberingHeaderLine + "0140;0140000000;0140999999;NEWO;M\xe9tropole;01/01/2017\r\n"
	snap, err := ParseNumbering(strings.NewReader(in), nil)
	if err != nil {
		t.Fatalf("ParseNumbering() error = %v", err)
	}
	want := []string{"0140", "0140000000", "0140999999", "NEWO", "", "", "Métropole", "2017-01-01"}
	if len(snap.Rows) != 1 || !slices.Equal(snap.Rows[0], want) {
		t.Errorf("Rows = %q, want [%q]", snap.Rows, want)
	}
}

func TestParseNumberingErrors(t *testing.T) {
	t.Parallel()
	row := func(fields string) string { return numberingHeaderLine + fields + "\r\n" }
	tests := map[string]string{
		"empty":               "",
		"UTF-8 header":        "EZABPQM;Tranche_Debut;Tranche_Fin;Mnémo;Territoire;Date_Attribution\r\n",
		"missing field":       row("0140;0140000000;0140999999;FRTE;M\xe9tropole"),
		"empty prefix":        row(";0140000000;0140999999;FRTE;M\xe9tropole;01/01/2017"),
		"scientific notation": row("0140;1,40E+08;0140999999;FRTE;M\xe9tropole;01/01/2017"),
		// As in the update of 11/04/2025, reported on data.gouv.fr.
		"leading zero lost":    row("3101;310100000;310199999;SFR0;M\xe9tropole;01/01/2017"),
		"national of 9 digits": row("0140;014000000;014099999;FRTE;M\xe9tropole;01/01/2017"),
		"short of 7 digits":    row("3101;3101000;3101999;SFR0;M\xe9tropole;01/01/2017"),
		"lengths differ":       row("0140;014000000;0140999999;FRTE;M\xe9tropole;01/01/2017"),
		"first after last":     row("0140;0140999999;0140000000;FRTE;M\xe9tropole;01/01/2017"),
		"outside its prefix":   row("0140;0141000000;0141999999;FRTE;M\xe9tropole;01/01/2017"),
		"empty operator":       row("0140;0140000000;0140999999;;M\xe9tropole;01/01/2017"),
		"empty territory":      row("0140;0140000000;0140999999;FRTE;;01/01/2017"),
		"ISO date":             row("0140;0140000000;0140999999;FRTE;M\xe9tropole;2017-01-01"),
		"overlapping ranges": numberingHeaderLine +
			"0140;0140000000;0140999999;FRTE;M\xe9tropole;01/01/2017\r\n" +
			"01409;0140900000;0140999999;SFR0;M\xe9tropole;01/01/2017\r\n",
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseNumbering(strings.NewReader(in), nil); !errors.Is(err, ErrFormat) {
				t.Errorf("ParseNumbering() error = %v, want %v", err, ErrFormat)
			}
		})
	}
}

func BenchmarkParseNumbering(b *testing.B) {
	in := readFixture(b, numberingFixture)
	ops := fixtureOperators(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParseNumbering(bytes.NewReader(in), ops); err != nil {
			b.Fatal(err)
		}
	}
}
