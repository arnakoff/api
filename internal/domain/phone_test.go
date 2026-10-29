// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

func TestParsePhone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		in           string
		wantE164     string
		wantCode     int
		wantNational string // empty: outside the French numbering plan
	}{
		{"national mobile", "06 12 34 56 78", "+33612345678", 33, "0612345678"},
		{"no separator", "0612345678", "+33612345678", 33, "0612345678"},
		{"dots", "06.12.34.56.78", "+33612345678", 33, "0612345678"},
		{"dashes", "06-12-34-56-78", "+33612345678", 33, "0612345678"},
		{"no-break spaces", "06 12 34 56 78", "+33612345678", 33, "0612345678"},
		{"narrow no-break spaces", "06 12 34 56 78", "+33612345678", 33, "0612345678"},
		{"surrounding spaces", " \t01 23 45 67 89 ", "+33123456789", 33, "0123456789"},
		{"plus country code", "+33 6 12 34 56 78", "+33612345678", 33, "0612345678"},
		{"international prefix", "0033 6 12 34 56 78", "+33612345678", 33, "0612345678"},
		{"bracketed trunk prefix", "+33 (0)6 12 34 56 78", "+33612345678", 33, "0612345678"},
		{"non-geographic", "0800 12 34 56", "+33800123456", 33, "0800123456"},
		{"shared overseas VoIP stays French", "0939 50 00 00", "+33939500000", 33, "0939500000"},
		{"Guadeloupe mobile", "0690 12 34 56", "+590690123456", 590, "0690123456"},
		{"Guadeloupe fixed", "0590 01 00 00", "+590590010000", 590, "0590010000"},
		{"Martinique fixed", "0596 00 00 00", "+596596000000", 596, "0596000000"},
		{"Martinique mobile", "0697 00 00 00", "+596697000000", 596, "0697000000"},
		{"Guyane mobile", "0694 01 00 00", "+594694010000", 594, "0694010000"},
		{"La Réunion fixed", "0262 01 00 00", "+262262010000", 262, "0262010000"},
		{"La Réunion mobile", "0693 00 00 00", "+262693000000", 262, "0693000000"},
		{"Mayotte fixed", "0269 61 23 45", "+262269612345", 262, "0269612345"},
		{"Mayotte mobile", "0639 01 00 00", "+262639010000", 262, "0639010000"},
		{"overseas number written with +33", "+33 6 90 12 34 56", "+590690123456", 590, "0690123456"},
		{"overseas number with its own code", "+590 690 12 34 56", "+590690123456", 590, "0690123456"},
		{"United Kingdom", "+44 20 7946 0958", "+442079460958", 44, ""},
		{"nine digits outside France", "+32 470 12 34 56", "+32470123456", 32, ""},
		{"United States", "001 202 555 0123", "+12025550123", 1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParsePhone(tt.in)
			if err != nil {
				t.Fatalf("ParsePhone(%q) error = %v", tt.in, err)
			}
			if got.String() != tt.wantE164 {
				t.Errorf("ParsePhone(%q).String() = %q, want %q", tt.in, got.String(), tt.wantE164)
			}
			if got.CountryCode() != tt.wantCode {
				t.Errorf("ParsePhone(%q).CountryCode() = %d, want %d", tt.in, got.CountryCode(), tt.wantCode)
			}
			national, ok := got.FrenchNational()
			if national != tt.wantNational || ok != (tt.wantNational != "") {
				t.Errorf("ParsePhone(%q).FrenchNational() = %q, %t; want %q", tt.in, national, ok, tt.wantNational)
			}
		})
	}
}

func TestParsePhoneRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"spaces only", "   "},
		{"letters", "06 12 34 AB CD"},
		{"vanity letters", "0800 FLOWER"},
		{"extension", "06 12 34 56 78 ext 12"},
		{"tel URI", "tel:+33612345678"},
		{"short number", "3631"},
		{"too short", "0612"},
		{"too long", "061234567890"},
		{"unknown country code", "+999 123 456 789"},
		{"plus only", "+"},
		{"long input", strings.Repeat("0", 65)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParsePhone(tt.in)
			if !errors.Is(err, domain.ErrInvalidPhone) {
				t.Fatalf("ParsePhone(%q) = %q, %v; want error %v", tt.in, got, err, domain.ErrInvalidPhone)
			}
		})
	}
}

// A phone number is personal data: a parse error must not quote it, since
// errors end up in logs.
func TestParsePhoneErrorDoesNotQuoteInput(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"06 12 34 56 78 ext 12", "061234567890", "+999 123 456 789", "0612 34 AB"} {
		_, err := domain.ParsePhone(in)
		if err == nil {
			t.Fatalf("ParsePhone(%q) error = nil, want an error", in)
		}
		for _, part := range []string{"12", "34", "56", "999", "AB"} {
			if strings.Contains(err.Error(), part) {
				t.Errorf("ParsePhone(%q) error %q quotes %q", in, err, part)
			}
		}
	}
}

func TestPhoneRedacted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"06 12 34 56 78", "+33***78"},
		{"0690 12 34 56", "+590***56"},
		{"+44 20 7946 0958", "+44***58"},
	}
	for _, tt := range tests {
		p, err := domain.ParsePhone(tt.in)
		if err != nil {
			t.Fatalf("ParsePhone(%q) error = %v", tt.in, err)
		}
		if got := p.Redacted(); got != tt.want {
			t.Errorf("ParsePhone(%q).Redacted() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func FuzzParsePhone(f *testing.F) {
	for _, s := range []string{
		"06 12 34 56 78", "+33 (0)6 12 34 56 78", "0033612345678", "0690123456", "+590 690 12 34 56",
		"+44 20 7946 0958", "06 12 34 56 78 ext 12", "3631", "+", "",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, err := domain.ParsePhone(s)
		if err != nil {
			if !errors.Is(err, domain.ErrInvalidPhone) {
				t.Fatalf("ParsePhone(%q) error = %v, want %v", s, err, domain.ErrInvalidPhone)
			}
			return
		}
		checkPhoneInvariants(t, s, p)
	})
}

// checkPhoneInvariants fails t if p, parsed from s, does not read back as
// itself or exposes a malformed national or redacted form.
func checkPhoneInvariants(t *testing.T, s string, p domain.Phone) {
	t.Helper()
	e164 := p.String()
	again, err := domain.ParsePhone(e164)
	if err != nil {
		t.Fatalf("ParsePhone(%q) = %q, which does not parse again: %v", s, e164, err)
	}
	if again != p {
		t.Fatalf("ParsePhone(%q) = %q, parsed again = %q", s, e164, again)
	}
	if national, ok := p.FrenchNational(); ok && (len(national) != 10 || national[0] != '0') {
		t.Fatalf("ParsePhone(%q).FrenchNational() = %q, want ten digits starting with 0", s, national)
	}
	if want := "+" + strconv.Itoa(p.CountryCode()) + "***" + e164[len(e164)-2:]; p.Redacted() != want {
		t.Fatalf("ParsePhone(%q).Redacted() = %q, want %q", s, p.Redacted(), want)
	}
}

func BenchmarkParsePhone(b *testing.B) {
	for _, in := range []string{"06 12 34 56 78", "+33 (0)6 12 34 56 78", "0690 12 34 56", "+44 20 7946 0958"} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := domain.ParsePhone(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
