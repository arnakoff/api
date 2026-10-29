// SPDX-License-Identifier: AGPL-3.0-only

package amf

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

// fixtures holds trimmed extracts of the real dataset metadata and file,
// downloaded on 29/10/2026.
var fixtures = os.DirFS("../../../../testdata/amf")

// csvFixture is the trimmed file: 27 entries, 4 of them free-text names.
const csvFixture = "listesnoires.csv"

func readFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	b, err := fs.ReadFile(fixtures, name)
	if err != nil {
		tb.Fatalf("read fixture: %v", err)
	}
	return b
}

func TestParseFixture(t *testing.T) {
	t.Parallel()
	snap, err := ParseBlacklist(bytes.NewReader(readFixture(t, csvFixture)))
	if err != nil {
		t.Fatalf("ParseBlacklist() error = %v", err)
	}
	if snap.Source != BlacklistSource {
		t.Errorf("BlacklistSource = %q, want %q", snap.Source, BlacklistSource)
	}
	if !slices.Equal(snap.Header, blacklistColumns) {
		t.Errorf("Header = %q, want %q", snap.Header, blacklistColumns)
	}
	// 27 entries, 4 free-text names left out.
	if len(snap.Rows) != 23 {
		t.Errorf("len(Rows) = %d, want 23", len(snap.Rows))
	}
	if want := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC); !snap.Newest.Equal(want) {
		t.Errorf("Newest = %v, want %v", snap.Newest, want)
	}
	for _, r := range snap.Rows {
		if strings.Contains(r[3], "Kuvera") || strings.Contains(r[3], "Plan B") {
			t.Errorf("free-text name kept: %q", r)
		}
	}
}

func TestParseRows(t *testing.T) {
	t.Parallel()
	snap, err := ParseBlacklist(bytes.NewReader(readFixture(t, csvFixture)))
	if err != nil {
		t.Fatalf("ParseBlacklist() error = %v", err)
	}
	tests := []struct {
		entry string
		want  []string
	}{
		{"prénom.nom@paris-majeste.com", []string{"email", "prénom.nom@paris-majeste.com", "", "prénom.nom@paris-majeste.com", "Usurpation", "2026-10-08"}},
		{"alphaairobot.com", []string{"domain", "alphaairobot.com", "", "alphaairobot.com", "Crypto-actifs|Usurpation|Usurpation_AMF", "2026-07-29"}},
		{"hlf-gestion-or.com", []string{"domain", "hlf-gestion-or.com", "", "hlf-gestion-or.com", "Biens_divers/Or|Autre_acteur_non_autorisé", "2026-06-12"}},
		{"calendly.com/amf-france", []string{"domain", "calendly.com", "/amf-france", "calendly.com/amf-france", "Usurpation_AMF", "2024-01-09"}},
		{"https://www.groupe-ytria.com", []string{"domain", "www.groupe-ytria.com", "", "https://www.groupe-ytria.com", "Usurpation", "2020-12-04"}},
	}
	for _, tt := range tests {
		i := slices.IndexFunc(snap.Rows, func(r []string) bool { return r[3] == tt.entry })
		if i < 0 {
			t.Errorf("no row for %q", tt.entry)
			continue
		}
		if got := snap.Rows[i]; !slices.Equal(got, tt.want) {
			t.Errorf("row %q = %q, want %q", tt.entry, got, tt.want)
		}
	}
}

func TestParseKeepsEveryListing(t *testing.T) {
	t.Parallel()
	snap, err := ParseBlacklist(bytes.NewReader(readFixture(t, csvFixture)))
	if err != nil {
		t.Fatalf("ParseBlacklist() error = %v", err)
	}
	var dates []string
	for _, r := range snap.Rows {
		if r[1] == "www.gestion.finance" {
			dates = append(dates, r[5])
		}
	}
	if want := []string{"2025-11-05", "2025-10-07"}; !slices.Equal(dates, want) {
		t.Errorf("listings of www.gestion.finance = %q, want %q", dates, want)
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	const header = "\xef\xbb\xbf\"nom\";\"categorie\";\"date_inscription\"\n"
	tests := []struct {
		name string
		in   string
		want error
	}{
		{"empty", "", ErrFormat},
		{"other header", "\"nom\";\"categories\";\"date_inscription\"\n", ErrFormat},
		{"comma separated", "nom,categorie,date_inscription\n", ErrFormat},
		{"missing field", header + "\"armen.ltd\";\"Usurpation\"\n", ErrFormat},
		{"French date", header + "\"armen.ltd\";\"Usurpation\";\"08/10/2026\"\n", ErrFormat},
		{"empty category", header + "\"armen.ltd\";\"\";\"2026-10-08\"\n", ErrFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseBlacklist(strings.NewReader(tt.in)); !errors.Is(err, tt.want) {
				t.Errorf("ParseBlacklist() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseErrorsLeaveValuesOut(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"address as header": `"contact@exemple.fr";"Usurpation";"2026-10-08"` + "\n",
		"address as date": "nom;categorie;date_inscription\n" +
			`"contact@exemple.fr";"Usurpation";"contact@exemple.fr"` + "\n",
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseBlacklist(strings.NewReader(in))
			if !errors.Is(err, ErrFormat) || strings.Contains(err.Error(), "exemple") {
				t.Errorf("ParseBlacklist() error = %v, want ErrFormat without the value", err)
			}
		})
	}
}

func TestParseWithoutBOM(t *testing.T) {
	t.Parallel()
	in := "\"nom\";\"categorie\";\"date_inscription\"\n\"armen.ltd\";\"Usurpation\";\"2026-10-08\"\n"
	snap, err := ParseBlacklist(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseBlacklist() error = %v", err)
	}
	if len(snap.Rows) != 1 {
		t.Errorf("len(Rows) = %d, want 1", len(snap.Rows))
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
