// SPDX-License-Identifier: AGPL-3.0-only

package acpr

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// fixtures holds a trimmed extract of the real file, downloaded on
// 30/10/2026.
var fixtures = os.DirFS("../../../../testdata/acpr")

// csvFixture is the trimmed file: 33 entries, 4 of them free-text names or
// entries with a space inside.
const csvFixture = "liste-noire.csv"

func readFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	b, err := fs.ReadFile(fixtures, name)
	if err != nil {
		tb.Fatalf("read fixture: %v", err)
	}
	return b
}

func parseFixture(t *testing.T) [][]string {
	t.Helper()
	snap, err := ParseBlacklist(bytes.NewReader(readFixture(t, csvFixture)))
	if err != nil {
		t.Fatalf("ParseBlacklist() error = %v", err)
	}
	return snap.Rows
}

func TestParseFixture(t *testing.T) {
	t.Parallel()
	snap, err := ParseBlacklist(bytes.NewReader(readFixture(t, csvFixture)))
	if err != nil {
		t.Fatalf("ParseBlacklist() error = %v", err)
	}
	if snap.Source != BlacklistSource {
		t.Errorf("Source = %q, want %q", snap.Source, BlacklistSource)
	}
	if !slices.Equal(snap.Header, blacklistColumns) {
		t.Errorf("Header = %q, want %q", snap.Header, blacklistColumns)
	}
	if len(snap.Rows) != 29 {
		t.Errorf("len(Rows) = %d, want 29", len(snap.Rows))
	}
	if want := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC); !snap.Newest.Equal(want) {
		t.Errorf("Newest = %v, want %v", snap.Newest, want)
	}
	// Free-text names, entries with a space inside and the Dénomination
	// column, which may name a natural person, are left out.
	for _, r := range snap.Rows {
		for _, left := range []string{"COSMOS", "Kuvera", "trader-ai.fr", "inversis-invest.com", "WETTEX BANK", "Automata"} {
			if strings.Contains(strings.Join(r, ","), left) {
				t.Errorf("row %q holds %q, which is left out", r, left)
			}
		}
	}
}

func TestParseRows(t *testing.T) {
	t.Parallel()
	rows := parseFixture(t)
	tests := []struct {
		entry string
		want  []string
	}{
		{"banque-stellantis.com", []string{"domain", "banque-stellantis.com", "", "banque-stellantis.com", "Crédits livrets paiements assurances|Usurpation Professionnel", "2026-10-26"}},
		{"prénom.nom@citieurope-private.com", []string{"email", "prénom.nom@citieurope-private.com", "", "prénom.nom@citieurope-private.com", "Crédits livrets paiements assurances|Usurpation Professionnel", "2026-10-26"}},
		{"info@wettex-bank.com", []string{"email", "info@wettex-bank.com", "", "info@wettex-bank.com", "Crédits livrets paiements assurances", "2026-09-02"}},
		{"login.easyinvest1.fr/login", []string{"domain", "login.easyinvest1.fr", "/login", "login.easyinvest1.fr/login", "Crédits livrets paiements assurances|Usurpation Professionnel|Biens divers : or", "2023-12-22"}},
		{"vantagemarkets.com/fr/", []string{"domain", "vantagemarkets.com", "/fr/", "vantagemarkets.com/fr/", "Forex", "2026-06-12"}},
		{"skda-management.com", []string{"domain", "skda-management.com", "", "skda-management.com", "Usurpation Professionnel", "2021-03-23"}},
		{"monplantfraise.fr", []string{"domain", "monplantfraise.fr", "", "monplantfraise.fr", "", "2024-09-30"}},
		{"bforex.com", []string{"domain", "bforex.com", "", "bforex.com", "Forex", "2010-12-01"}},
	}
	for _, tt := range tests {
		i := slices.IndexFunc(rows, func(r []string) bool { return r[3] == tt.entry })
		if i < 0 {
			t.Errorf("no row for %q", tt.entry)
			continue
		}
		if got := rows[i]; !slices.Equal(got, tt.want) {
			t.Errorf("row of %q = %q, want %q", tt.entry, got, tt.want)
		}
	}
}

// TestParseKeepsEveryListing checks that an entry listed twice, once
// without a category, gives two rows, as the AMF connector does.
func TestParseKeepsEveryListing(t *testing.T) {
	t.Parallel()
	var cats []string
	for _, r := range parseFixture(t) {
		if r[1] == "stabletrader-ai.net" {
			cats = append(cats, r[4])
		}
	}
	if want := []string{"", "Produits dérivés sur crypto-actifs"}; !slices.Equal(cats, want) {
		t.Errorf("categories of stabletrader-ai.net = %q, want %q", cats, want)
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	const header = "\xef\xbb\xbf\"URL / Mails\",Dénomination,Catégorie(s),\"Date d'inscription\"\n"
	tests := map[string]string{
		"empty":         "",
		"html page":     "<html></html>\n",
		"semicolons":    "\"URL / Mails\";Dénomination;Catégorie(s);\"Date d'inscription\"\n",
		"missing field": header + "armen.ltd,,Forex\n",
		"iso date":      header + "armen.ltd,,Forex,2026-10-13\n",
		"empty date":    header + "armen.ltd,,Forex,\n",
		"no such day":   header + "contact@exemple.fr,,Forex,31/02/2026\n",
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseBlacklist(strings.NewReader(in))
			if !errors.Is(err, ErrFormat) {
				t.Fatalf("ParseBlacklist() error = %v, want ErrFormat", err)
			}
			if strings.Contains(err.Error(), "exemple") {
				t.Errorf("error %q quotes a value of the file", err)
			}
		})
	}
}

func BenchmarkParseBlacklist(b *testing.B) {
	in := readFixture(b, csvFixture)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParseBlacklist(bytes.NewReader(in)); err != nil {
			b.Fatal(err)
		}
	}
}
