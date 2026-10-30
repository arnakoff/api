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

// psanFixture is the trimmed PSAN file: 17 rows, 12 providers.
const psanFixture = "psan.csv"

func TestParsePSANFixture(t *testing.T) {
	t.Parallel()
	snap, err := ParsePSAN(bytes.NewReader(readFixture(t, psanFixture)))
	if err != nil {
		t.Fatalf("ParsePSAN() error = %v", err)
	}
	if snap.Source != PSANSource || !slices.Equal(snap.Header, psanColumns) {
		t.Errorf("ParsePSAN() = %s %q, want %s %q", snap.Source, snap.Header, PSANSource, psanColumns)
	}
	// One row per key of a provider: activity rows of the same key merge.
	if len(snap.Rows) != 28 {
		t.Errorf("len(Rows) = %d, want 28", len(snap.Rows))
	}
	if want := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC); !snap.Newest.Equal(want) {
		t.Errorf("Newest = %v, want %v", snap.Newest, want)
	}
}

func TestParsePSANRows(t *testing.T) {
	t.Parallel()
	snap, err := ParsePSAN(bytes.NewReader(readFixture(t, psanFixture)))
	if err != nil {
		t.Fatalf("ParsePSAN() error = %v", err)
	}
	coinhouse := []string{"E2020-001", "COINHOUSE SAS", "France", "Enregistrement", "Radié", "2020-03-17", "2026-05-07"}
	luno := []string{"E2022-045", "LUNO FRANCE SAS", "France", "Enregistrement", "Radié", "2022-07-19", "2025-07-28"}
	foris := []string{"N/A - Passeport", "FORIS DAX MT LIMITED", "Malte", "Agrément MICA", "Agréé", "2025-02-12", ""}
	paytop := []string{"A2026-015", "PAYTOP SAS (TRIPLE-A)", "France", "Agrément MICA", "Agréé", "2026-05-21", ""}
	want := [][]string{
		append([]string{"domain", "www.coinhouse.com", "/fr/", "https://www.coinhouse.com/fr/"}, coinhouse...),
		append([]string{"email", "support@coinhouse.com", "", "support@coinhouse.com"}, coinhouse...),
		append([]string{"phone", "+33182882951", "", "+33 1 82 88 29 51"}, coinhouse...),
		append([]string{"siren", "815254545", "", "815254545"}, coinhouse...),
		append([]string{"phone", "+33176360518", "", "+33 1 76 36 05 18"}, luno...),
		append([]string{"domain", "crypto.com", "/fr", "https://crypto.com/fr"}, foris...),
		append([]string{"domain", "www.paytop.com", "", "https://www.paytop.com/"}, paytop...),
		append([]string{"domain", "www.triple-a.io", "", "https://www.triple-a.io/"}, paytop...),
	}
	for _, w := range want {
		if n := countRows(snap.Rows, w); n != 1 {
			t.Errorf("row %q found %d times, want once", w, n)
		}
	}
}

func TestParsePSANSkipsWhatIsNotAKey(t *testing.T) {
	t.Parallel()
	snap, err := ParsePSAN(bytes.NewReader(readFixture(t, psanFixture)))
	if err != nil {
		t.Fatalf("ParsePSAN() error = %v", err)
	}
	for _, r := range snap.Rows {
		switch {
		case r[0] == "email" && strings.HasPrefix(r[3], "https://"):
			t.Errorf("contact page kept as an address: %q", r)
		case r[0] == "siren" && r[6] != "France":
			t.Errorf("foreign registry number kept as a SIREN: %q", r)
		}
	}
}

func TestParsePSANErrors(t *testing.T) {
	t.Parallel()
	header := strings.Join(quoteAll(psanHeader), ";") + "\n"
	row := func(change func([]string)) string {
		r := []string{
			"E2020-001", "COINHOUSE SAS", "", "France", "https://www.coinhouse.com/fr/", "", "", "815254545", "",
			"Enregistrement", "2020-03-17", "2026-05-07", "", "Radié", "Service sur actifs numériques",
			"Conservation d'actifs numériques", "2020-03-17", "2026-05-07", "2026-10-29",
		}
		change(r)
		return strings.Join(quoteAll(r), ";") + "\n"
	}
	tests := []struct {
		name string
		in   string
	}{
		{"blacklist file", "\"nom\";\"categorie\";\"date_inscription\"\n"},
		{"no AMF number", header + row(func(r []string) { r[0] = "" })},
		{"no name", header + row(func(r []string) { r[1] = "" })},
		{"unknown status", header + row(func(r []string) { r[13] = "Suspendu" })},
		{"no authorisation", header + row(func(r []string) { r[9] = "" })},
		{"French start date", header + row(func(r []string) { r[10] = "17/03/2020" })},
		{"French end date", header + row(func(r []string) { r[11] = "07/05/2026" })},
		{"withdrawn without an end date", header + row(func(r []string) { r[11] = "" })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParsePSAN(strings.NewReader(tt.in)); !errors.Is(err, ErrFormat) {
				t.Errorf("ParsePSAN() error = %v, want ErrFormat", err)
			}
		})
	}
}

func BenchmarkParsePSAN(b *testing.B) {
	in := readFixture(b, psanFixture)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParsePSAN(bytes.NewReader(in)); err != nil {
			b.Fatal(err)
		}
	}
}

// countRows returns how many rows equal want.
func countRows(rows [][]string, want []string) int {
	n := 0
	for _, r := range rows {
		if slices.Equal(r, want) {
			n++
		}
	}
	return n
}

// quoteAll returns the fields between double quotes, as the AMF writes them.
func quoteAll(fields []string) []string {
	q := make([]string, len(fields))
	for i, f := range fields {
		q[i] = `"` + f + `"`
	}
	return q
}
