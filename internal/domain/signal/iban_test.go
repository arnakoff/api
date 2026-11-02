// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"strings"
	"testing"

	"github.com/jacoelho/banking/iban"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

const ibanRegistryURL = "https://www.swift.com/standards/data-standards/iban-international-bank-account-number"

func parseIBAN(t testing.TB, s string) domain.IBAN {
	t.Helper()
	i, err := domain.ParseIBAN(s)
	if err != nil {
		t.Fatalf("ParseIBAN(%q): %v", s, err)
	}
	return i
}

func TestIBANCountrySignal(t *testing.T) {
	t.Parallel()

	// The example IBANs of the SWIFT IBAN registry.
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"France", "FR14 2004 1010 0505 0001 3M02 606", "Pays de tenue du compte : France (code FR de l'IBAN)."},
		{"Lithuania", "LT121000011101001000", "Pays de tenue du compte : Lituanie (code LT de l'IBAN)."},
		{"United Kingdom", "GB29NWBK60161331926819", "Pays de tenue du compte : Royaume-Uni (code GB de l'IBAN)."},
		{"United Arab Emirates", "AE070331234567890123456", "Pays de tenue du compte : Émirats arabes unis (code AE de l'IBAN)."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := signal.IBANCountry(parseIBAN(t, tt.in), checkedAt)
			want := domain.Signal{
				Code:       "iban_country",
				Level:      domain.LevelInfo,
				Message:    tt.want,
				Source:     "IBAN",
				SourceURL:  ibanRegistryURL,
				ObservedAt: checkedAt,
			}
			if !ok || got != want {
				t.Errorf("IBANCountry() = %+v, %v; want %+v, true", got, ok, want)
			}
		})
	}
}

// TestIBANCountrySignalHoldsOnlyTheCountry checks that the signal, which is
// published and may be logged, says nothing of the account but its country.
func TestIBANCountrySignalHoldsOnlyTheCountry(t *testing.T) {
	t.Parallel()

	in := parseIBAN(t, "FR1420041010050500013M02606")
	s, _ := signal.IBANCountry(in, checkedAt)

	for _, field := range []string{s.Code, s.Message, s.Advice, string(s.Source), s.SourceURL} {
		for _, raw := range []string{in.String(), in.String()[2:4], "20041", "13M02606", in.Redacted()} {
			if strings.Contains(field, raw) {
				t.Errorf("signal field %q holds %q", field, raw)
			}
		}
	}
}

// TestIBANCountryNamesEveryRegistryCountry checks that every country the
// IBAN registry knows has a French name, so no valid IBAN goes without its
// signal.
func TestIBANCountryNamesEveryRegistryCountry(t *testing.T) {
	t.Parallel()

	countries := 0
	for a := 'A'; a <= 'Z'; a++ {
		for b := 'A'; b <= 'Z'; b++ {
			code := string([]rune{a, b})
			generated, err := iban.Generate(code)
			if err != nil {
				continue // not an IBAN country
			}
			countries++
			s, ok := signal.IBANCountry(parseIBAN(t, generated), checkedAt)
			if !ok || strings.Contains(s.Message, ": "+code+" (") {
				t.Errorf("%s: IBANCountry() = %q, %v; want a French country name", code, s.Message, ok)
			}
		}
	}
	if countries < 80 {
		t.Errorf("found %d IBAN countries, want the whole registry", countries)
	}
}

func BenchmarkIBANCountry(b *testing.B) {
	in := parseIBAN(b, "LT121000011101001000")
	b.ReportAllocs()
	for b.Loop() {
		signal.IBANCountry(in, checkedAt)
	}
}

func TestIBANInfoNamesTheRegistryTheSignalCites(t *testing.T) {
	t.Parallel()

	info := signal.IBANInfo()

	if info.Name != signal.IBANSource || info.URL != ibanRegistryURL || info.Publisher == "" || info.Licence == "" {
		t.Errorf("IBANInfo() = %+v, want %s at %s with a publisher and a licence", info, signal.IBANSource, ibanRegistryURL)
	}
}
