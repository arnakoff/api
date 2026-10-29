// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

// The SIRET numbers below are real establishments of the public bodies named
// in siren_test.go, read from the SIRENE register.

func TestParseSIRET(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    string
		want  string
		siren string
		nic   string
	}{
		{"digits", "12002701600563", "12002701600563", "120027016", "00563"},
		{"paper form", "572 104 891 00013", "57210489100013", "572104891", "00013"},
		{"no-break spaces", "519 037 584 14786", "51903758414786", "519037584", "14786"},
		{"surrounding spaces", " \t12002701600563\n", "12002701600563", "120027016", "00563"},
		// La Poste establishments fail the Luhn check but their digits add up
		// to a multiple of 5; its head office passes the Luhn check.
		{"La Poste head office", "35600000000048", "35600000000048", "356000000", "00048"},
		{"La Poste establishment", "356 000 000 09075", "35600000009075", "356000000", "09075"},
		{"another La Poste establishment", "35600000067814", "35600000067814", "356000000", "67814"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseSIRET(tt.in)
			if err != nil {
				t.Fatalf("ParseSIRET(%q) error = %v", tt.in, err)
			}
			if got.String() != tt.want {
				t.Errorf("ParseSIRET(%q).String() = %q, want %q", tt.in, got.String(), tt.want)
			}
			if got.SIREN().String() != tt.siren {
				t.Errorf("ParseSIRET(%q).SIREN() = %q, want %q", tt.in, got.SIREN(), tt.siren)
			}
			if got.NIC() != tt.nic {
				t.Errorf("ParseSIRET(%q).NIC() = %q, want %q", tt.in, got.NIC(), tt.nic)
			}
		})
	}
}

func TestParseSIRETRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"spaces only", "   "},
		{"wrong check digit", "12002701600564"},
		// Luhn-valid as a whole, but its SIREN part is not a valid SIREN.
		{"invalid SIREN part", "12002701700561"},
		// Digit sum a multiple of 5, but not a La Poste number.
		{"digit sum rule outside La Poste", "12002701600565"},
		{"La Poste typo", "35600000009076"},
		{"thirteen digits", "1200270160056"},
		{"SIREN", "120027016"},
		{"dots", "120.027.016.00563"},
		{"letter", "1200270160056A"},
		{"long input", strings.Repeat(" ", 40) + "12002701600563"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseSIRET(tt.in)
			if !errors.Is(err, domain.ErrInvalidSIRET) {
				t.Fatalf("ParseSIRET(%q) = %q, %v; want error %v", tt.in, got, err, domain.ErrInvalidSIRET)
			}
		})
	}
}

func FuzzParseSIRET(f *testing.F) {
	for _, s := range []string{
		"12002701600563", "572 104 891 00013", "35600000009075", "35600000000048",
		"12002701600564", "12002701700561", "120027016", "",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		n, err := domain.ParseSIRET(s)
		if err != nil {
			if !errors.Is(err, domain.ErrInvalidSIRET) {
				t.Fatalf("ParseSIRET(%q) error = %v, want %v", s, err, domain.ErrInvalidSIRET)
			}
			return
		}
		checkSIRETInvariants(t, s, n)
	})
}

// checkSIRETInvariants fails t if n, parsed from s, does not read back as
// itself or does not split into a valid SIREN and a five-digit NIC.
func checkSIRETInvariants(t *testing.T, s string, n domain.SIRET) {
	t.Helper()
	digits := n.String()
	if len(digits) != 14 || strings.Trim(digits, "0123456789") != "" {
		t.Fatalf("ParseSIRET(%q).String() = %q, want fourteen digits", s, digits)
	}
	again, err := domain.ParseSIRET(digits)
	if err != nil || again != n {
		t.Fatalf("ParseSIRET(%q) = %q, parsed again = %q, %v", s, digits, again, err)
	}
	siren, err := domain.ParseSIREN(digits[:9])
	if err != nil || n.SIREN() != siren {
		t.Fatalf("ParseSIRET(%q).SIREN() = %q, want %q, %v", s, n.SIREN(), digits[:9], err)
	}
	if n.NIC() != digits[9:] {
		t.Fatalf("ParseSIRET(%q).NIC() = %q, want %q", s, n.NIC(), digits[9:])
	}
}

func BenchmarkParseSIRET(b *testing.B) {
	for _, in := range []string{"12002701600563", "572 104 891 00013", "35600000009075"} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := domain.ParseSIRET(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
