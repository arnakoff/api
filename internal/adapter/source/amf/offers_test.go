// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// offersFixture is the trimmed biens divers file: 7 offers, 3 of them with
// neither site nor phone.
const offersFixture = "biens-divers.csv"

func TestParseOffersFixture(t *testing.T) {
	t.Parallel()
	snap, err := ParseOffers(bytes.NewReader(readFixture(t, offersFixture)))
	if err != nil {
		t.Fatalf("ParseOffers() error = %v", err)
	}
	if snap.Source != OffersSource || !slices.Equal(snap.Header, offersColumns) {
		t.Errorf("ParseOffers() = %s %q, want %s %q", snap.Source, snap.Header, OffersSource, offersColumns)
	}
	if len(snap.Rows) != 8 {
		t.Errorf("len(Rows) = %d, want 8", len(snap.Rows))
	}
	if want := time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC); !snap.Newest.Equal(want) {
		t.Errorf("Newest = %v, want %v", snap.Newest, want)
	}
	bergere := []string{"D-14-01", "LA BERGERE : INVESTISSEMENT DANS LES GRANDS CRUS", "2014-07-23"}
	want := [][]string{
		append([]string{
			"domain", "la-bergere.fr", "/groupe-labergere/offre-la-bergere-investment/",
			"https://la-bergere.fr/groupe-labergere/offre-la-bergere-investment/",
		}, bergere...),
		append([]string{"phone", "+33557934159", "", "+33 (0)5 57 93 41 59"}, bergere...),
		{"domain", "u.wine", "", "https://u.wine/", "D-14-02", "U’WINEVEST", "2014-08-26"},
	}
	for _, w := range want {
		if n := countRows(snap.Rows, w); n != 1 {
			t.Errorf("row %q found %d times, want once", w, n)
		}
	}
}

func TestParseOffersErrors(t *testing.T) {
	t.Parallel()
	header := strings.Join(quoteAll(offersHeader), ";") + "\n"
	row := func(change func([]string)) string {
		r := []string{"D-14-02", "U’WINEVEST", "https://u.wine/", "+33 (0)5 35 54 61 39", "2014-08-26", "2026-10-29"}
		change(r)
		return strings.Join(quoteAll(r), ";") + "\n"
	}
	tests := []struct {
		name string
		in   string
	}{
		{"PSAN file", strings.Join(quoteAll(psanHeader), ";") + "\n"},
		{"no AMF number", header + row(func(r []string) { r[0] = "" })},
		{"no name", header + row(func(r []string) { r[1] = "" })},
		{"French date", header + row(func(r []string) { r[4] = "26/08/2014" })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseOffers(strings.NewReader(tt.in)); !errors.Is(err, ErrFormat) {
				t.Errorf("ParseOffers() error = %v, want ErrFormat", err)
			}
		})
	}
}
