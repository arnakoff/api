// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

func TestParseInputDetects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		kind domain.Kind
		want string
	}{
		{"domain", "exemple.fr", domain.KindDomain, "exemple.fr"},
		{"URL", "https://www.exemple.fr/contact?ref=1", domain.KindDomain, "www.exemple.fr"},
		// Credentials before the host are dropped: this is a URL, not an email.
		{"URL with credentials", "https://amf-france.org@exemple.com", domain.KindDomain, "exemple.com"},
		{"email", "contact@exemple.fr", domain.KindEmail, "contact@exemple.fr"},
		{"email with display name", "Service client <contact@exemple.fr>", domain.KindEmail, "contact@exemple.fr"},
		// Ten digits starting with 0: a French number, never a SIREN.
		{"national phone", "0612345678", domain.KindPhone, "+33612345678"},
		{"grouped phone", "06 12 34 56 78", domain.KindPhone, "+33612345678"},
		{"international phone", "+33 6 12 34 56 78", domain.KindPhone, "+33612345678"},
		{"dotted phone", "06.12.34.56.78", domain.KindPhone, "+33612345678"},
		// Fourteen digits that are not a SIRET: an international number.
		{"fourteen-digit phone", "0044 20 7946 0958", domain.KindPhone, "+442079460958"},
		// Two letters then two digits: an IBAN.
		{"IBAN", "FR1420041010050500013M02606", domain.KindIBAN, "FR1420041010050500013M02606"},
		{"IBAN paper form", "fr14 2004 1010 0505 0001 3m02 606", domain.KindIBAN, "FR1420041010050500013M02606"},
		// Nine digits: a SIREN.
		{"SIREN", "120027016", domain.KindSIREN, "120027016"},
		{"grouped SIREN", "120 027 016", domain.KindSIREN, "120027016"},
		{"SIRET", "120 027 016 00563", domain.KindSIRET, "12002701600563"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseInput(tt.in, domain.KindAuto)
			if err != nil {
				t.Fatalf("ParseInput(%q) error = %v", tt.in, err)
			}
			if got.Kind() != tt.kind || got.String() != tt.want {
				t.Errorf("ParseInput(%q) = %v %q, want %v %q", tt.in, got.Kind(), got, tt.kind, tt.want)
			}
		})
	}
}

func TestParseInputHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		hint domain.Kind
		want string
	}{
		{"nine digits as a phone", "612345678", domain.KindPhone, "+33612345678"},
		{"email as a domain", "contact@exemple.fr", domain.KindDomain, "exemple.fr"},
		{"SIRET as a SIRET", "12002701600563", domain.KindSIRET, "12002701600563"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseInput(tt.in, tt.hint)
			if err != nil {
				t.Fatalf("ParseInput(%q, %v) error = %v", tt.in, tt.hint, err)
			}
			if got.Kind() != tt.hint || got.String() != tt.want {
				t.Errorf("ParseInput(%q, %v) = %v %q, want %q", tt.in, tt.hint, got.Kind(), got, tt.want)
			}
		})
	}
}

func TestParseInputRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    string
		hint  domain.Kind
		cause error
	}{
		{"empty", "", domain.KindAuto, nil},
		{"spaces only", " \t\n", domain.KindAuto, nil},
		{"unknown hint", "exemple.fr", domain.Kind(99), nil},
		// Nine digits stay a SIREN even with a wrong check digit.
		{"SIREN typo", "120027017", domain.KindAuto, domain.ErrInvalidSIREN},
		{"IBAN typo", "FR1520041010050500013M02606", domain.KindAuto, domain.ErrInvalidIBAN},
		{"fourteen digits, neither SIRET nor phone", "12002701600564", domain.KindAuto, domain.ErrInvalidSIRET},
		{"short number", "3615", domain.KindAuto, domain.ErrInvalidPhone},
		{"bad email", "contact@exemple", domain.KindAuto, domain.ErrInvalidEmail},
		{"word", "arnaque", domain.KindAuto, domain.ErrInvalidDomain},
		{"FTP URL", "ftp://exemple.fr", domain.KindAuto, domain.ErrInvalidDomain},
		{"hint mismatch", "exemple.fr", domain.KindIBAN, domain.ErrInvalidIBAN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseInput(tt.in, tt.hint)
			if !errors.Is(err, domain.ErrUnrecognisedInput) {
				t.Fatalf("ParseInput(%q, %v) = %v, %v; want error %v", tt.in, tt.hint, got, err, domain.ErrUnrecognisedInput)
			}
			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Errorf("ParseInput(%q, %v) error = %v, want it to wrap %v", tt.in, tt.hint, err, tt.cause)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind domain.Kind
		want string
	}{
		{domain.KindAuto, "auto"},
		{domain.KindDomain, "domain"},
		{domain.KindEmail, "email"},
		{domain.KindPhone, "phone"},
		{domain.KindIBAN, "iban"},
		{domain.KindSIREN, "siren"},
		{domain.KindSIRET, "siret"},
		{domain.Kind(99), "Kind(99)"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", uint8(tt.kind), got, tt.want)
		}
	}
}

func FuzzParseInput(f *testing.F) {
	for _, s := range []string{
		"exemple.fr", "https://amf-france.org@exemple.com", "Service client <contact@exemple.fr>",
		"06 12 34 56 78", "0044 20 7946 0958", "fr14 2004 1010 0505 0001 3m02 606",
		"120 027 016", "12002701600563", "120027017", "3615", "",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		in, err := domain.ParseInput(s, domain.KindAuto)
		if err != nil {
			if !errors.Is(err, domain.ErrUnrecognisedInput) {
				t.Fatalf("ParseInput(%q) error = %v, want %v", s, err, domain.ErrUnrecognisedInput)
			}
			return
		}
		// The normalised form is detected as the same input, with or without
		// its kind as a hint.
		for _, hint := range []domain.Kind{domain.KindAuto, in.Kind()} {
			again, err := domain.ParseInput(in.String(), hint)
			if err != nil || again != in {
				t.Fatalf("ParseInput(%q) = %v %q, parsed again with hint %v = %v, %v", s, in.Kind(), in, hint, again, err)
			}
		}
	})
}

func BenchmarkParseInput(b *testing.B) {
	for _, in := range []string{
		"https://www.exemple.fr/contact", "contact@exemple.fr", "06 12 34 56 78",
		"FR14 2004 1010 0505 0001 3M02 606", "120 027 016",
	} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := domain.ParseInput(in, domain.KindAuto); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestParseKind(t *testing.T) {
	t.Parallel()

	for _, want := range []domain.Kind{
		domain.KindDomain, domain.KindEmail, domain.KindPhone,
		domain.KindIBAN, domain.KindSIREN, domain.KindSIRET,
	} {
		t.Run(want.String(), func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseKind(want.String())
			if err != nil || got != want {
				t.Errorf("ParseKind(%q) = %v, %v; want %v, nil", want.String(), got, err, want)
			}
		})
	}
}

func TestParseKindRejects(t *testing.T) {
	t.Parallel()

	// "auto" is no kind: a corpus or a request names the kind it means.
	for _, s := range []string{"", "auto", "Domain", "url", "Kind(9)"} {
		if _, err := domain.ParseKind(s); !errors.Is(err, domain.ErrUnknownKind) {
			t.Errorf("ParseKind(%q) error = %v, want ErrUnknownKind", s, err)
		}
	}
}
