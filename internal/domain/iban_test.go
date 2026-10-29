// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

// The IBANs below are the examples of the SWIFT IBAN registry, also used by
// the tests of github.com/jacoelho/banking: they belong to no real account.

func TestParseIBAN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		in       string
		want     string
		country  string
		bankCode string // empty: not a French IBAN
	}{
		{"electronic form", "FR1420041010050500013M02606", "FR1420041010050500013M02606", "FR", "20041"},
		{"paper form", "FR14 2004 1010 0505 0001 3M02 606", "FR1420041010050500013M02606", "FR", "20041"},
		{"lower case", "fr14 2004 1010 0505 0001 3m02 606", "FR1420041010050500013M02606", "FR", "20041"},
		{"no-break spaces", "FR14 2004 1010 0505 0001 3M02 606", "FR1420041010050500013M02606", "FR", "20041"},
		{"surrounding spaces", " \tFR1420041010050500013M02606\n", "FR1420041010050500013M02606", "FR", "20041"},
		{"Germany", "DE89 3704 0044 0532 0130 00", "DE89370400440532013000", "DE", ""},
		{"United Kingdom", "GB82 WEST 1234 5698 7654 32", "GB82WEST12345698765432", "GB", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseIBAN(tt.in)
			if err != nil {
				t.Fatalf("ParseIBAN(%q) error = %v", tt.in, err)
			}
			if got.String() != tt.want {
				t.Errorf("ParseIBAN(%q).String() = %q, want %q", tt.in, got.String(), tt.want)
			}
			if got.Country() != tt.country {
				t.Errorf("ParseIBAN(%q).Country() = %q, want %q", tt.in, got.Country(), tt.country)
			}
			code, ok := got.FrenchBankCode()
			if code != tt.bankCode || ok != (tt.bankCode != "") {
				t.Errorf("ParseIBAN(%q).FrenchBankCode() = %q, %t; want %q", tt.in, code, ok, tt.bankCode)
			}
		})
	}
}

func TestParseIBANRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"spaces only", "   "},
		{"wrong check digits", "FR1520041010050500013M02606"},
		{"wrong length", "FR1420041010050500013M0260"},
		{"unknown country", "ZZ1420041010050500013M02606"},
		{"dashes", "FR14-2004-1010-0505-0001-3M02-606"},
		{"punctuation", "FR14.2004.1010.0505.0001.3M02.606"},
		{"non-ASCII letter", "FR1420041010050500013É02606"},
		{"long input", strings.Repeat("0", 65)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseIBAN(tt.in)
			if !errors.Is(err, domain.ErrInvalidIBAN) {
				t.Fatalf("ParseIBAN(%q) = %q, %v; want error %v", tt.in, got, err, domain.ErrInvalidIBAN)
			}
		})
	}
}

// An IBAN is personal data: a parse error must not quote it, since errors end
// up in logs.
func TestParseIBANErrorDoesNotQuoteInput(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"FR1520041010050500013M02606", "ZZ1420041010050500013M02606", "FR14-2004-1010"} {
		_, err := domain.ParseIBAN(in)
		if err == nil {
			t.Fatalf("ParseIBAN(%q) error = nil, want an error", in)
		}
		for _, part := range []string{"FR", "ZZ", "2004", "1010", "606"} {
			if strings.Contains(err.Error(), part) {
				t.Errorf("ParseIBAN(%q) error %q quotes %q", in, err, part)
			}
		}
	}
}

func TestIBANRedacted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"FR14 2004 1010 0505 0001 3M02 606", "FR***2606"},
		{"DE89 3704 0044 0532 0130 00", "DE***3000"},
	}
	for _, tt := range tests {
		i, err := domain.ParseIBAN(tt.in)
		if err != nil {
			t.Fatalf("ParseIBAN(%q) error = %v", tt.in, err)
		}
		if got := i.Redacted(); got != tt.want {
			t.Errorf("ParseIBAN(%q).Redacted() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func FuzzParseIBAN(f *testing.F) {
	for _, s := range []string{
		"FR1420041010050500013M02606", "fr14 2004 1010 0505 0001 3m02 606", "DE89370400440532013000",
		"GB82 WEST 1234 5698 7654 32", "FR1520041010050500013M02606", "FR14-2004", "",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		i, err := domain.ParseIBAN(s)
		if err != nil {
			if !errors.Is(err, domain.ErrInvalidIBAN) {
				t.Fatalf("ParseIBAN(%q) error = %v, want %v", s, err, domain.ErrInvalidIBAN)
			}
			return
		}
		checkIBANInvariants(t, s, i)
	})
}

// checkIBANInvariants fails t if i, parsed from s, does not read back as
// itself or exposes a malformed country, bank code or redacted form.
func checkIBANInvariants(t *testing.T, s string, i domain.IBAN) {
	t.Helper()
	electronic := i.String()
	again, err := domain.ParseIBAN(electronic)
	if err != nil || again != i {
		t.Fatalf("ParseIBAN(%q) = %q, parsed again = %q, %v", s, electronic, again, err)
	}
	if i.Country() != electronic[:2] {
		t.Fatalf("ParseIBAN(%q).Country() = %q, want %q", s, i.Country(), electronic[:2])
	}
	if code, ok := i.FrenchBankCode(); ok && (i.Country() != "FR" || code != electronic[4:9]) {
		t.Fatalf("ParseIBAN(%q).FrenchBankCode() = %q, want %q", s, code, electronic[4:9])
	}
	if want := electronic[:2] + "***" + electronic[len(electronic)-4:]; i.Redacted() != want {
		t.Fatalf("ParseIBAN(%q).Redacted() = %q, want %q", s, i.Redacted(), want)
	}
}

func BenchmarkParseIBAN(b *testing.B) {
	for _, in := range []string{"FR1420041010050500013M02606", "FR14 2004 1010 0505 0001 3M02 606", "GB82 WEST 1234 5698 7654 32"} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := domain.ParseIBAN(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
