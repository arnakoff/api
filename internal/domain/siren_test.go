// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

// The SIREN and SIRET numbers below are real public bodies, read from the
// SIRENE register through recherche-entreprises.api.gouv.fr: INSEE
// (120027016), Banque de France (572104891), SNCF Voyageurs (519037584) and
// La Poste (356000000).

func TestParseSIREN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"digits", "120027016", "120027016"},
		{"groups of three", "572 104 891", "572104891"},
		{"no-break spaces", "519 037 584", "519037584"},
		{"surrounding spaces", " \t356000000\n", "356000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseSIREN(tt.in)
			if err != nil {
				t.Fatalf("ParseSIREN(%q) error = %v", tt.in, err)
			}
			if got.String() != tt.want {
				t.Errorf("ParseSIREN(%q).String() = %q, want %q", tt.in, got.String(), tt.want)
			}
		})
	}
}

func TestParseSIRENRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"spaces only", "   "},
		{"wrong check digit", "120027017"},
		{"eight digits", "12002701"},
		{"ten digits", "1200270160"},
		{"SIRET", "12002701600563"},
		{"dots", "120.027.016"},
		{"letter", "12002701A"},
		{"non-ASCII digit", "12002701٦"},
		{"long input", strings.Repeat(" ", 40) + "120027016"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseSIREN(tt.in)
			if !errors.Is(err, domain.ErrInvalidSIREN) {
				t.Fatalf("ParseSIREN(%q) = %q, %v; want error %v", tt.in, got, err, domain.ErrInvalidSIREN)
			}
		})
	}
}

func FuzzParseSIREN(f *testing.F) {
	for _, s := range []string{"120027016", "572 104 891", "120027017", "12002701600563", "120.027.016", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		n, err := domain.ParseSIREN(s)
		if err != nil {
			if !errors.Is(err, domain.ErrInvalidSIREN) {
				t.Fatalf("ParseSIREN(%q) error = %v, want %v", s, err, domain.ErrInvalidSIREN)
			}
			return
		}
		digits := n.String()
		if len(digits) != 9 || strings.Trim(digits, "0123456789") != "" {
			t.Fatalf("ParseSIREN(%q).String() = %q, want nine digits", s, digits)
		}
		again, err := domain.ParseSIREN(digits)
		if err != nil || again != n {
			t.Fatalf("ParseSIREN(%q) = %q, parsed again = %q, %v", s, digits, again, err)
		}
	})
}

func BenchmarkParseSIREN(b *testing.B) {
	for _, in := range []string{"120027016", "572 104 891"} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := domain.ParseSIREN(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
