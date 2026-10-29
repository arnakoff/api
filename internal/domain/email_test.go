// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

func TestParseEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         string
		wantEmail  string
		wantLocal  string
		wantDomain string
	}{
		{"plain address", "contact@exemple.fr", "contact@exemple.fr", "contact", "exemple.fr"},
		{"surrounding spaces", "  contact@exemple.fr \t", "contact@exemple.fr", "contact", "exemple.fr"},
		{"display name dropped", "Service client <contact@exemple.fr>", "contact@exemple.fr", "contact", "exemple.fr"},
		{"encoded display name dropped", "=?utf-8?q?J=C3=A9r=C3=B4me?= <contact@exemple.fr>", "contact@exemple.fr", "contact", "exemple.fr"},
		{"angle brackets only", "<contact@exemple.fr>", "contact@exemple.fr", "contact", "exemple.fr"},
		{"comment dropped", "contact@exemple.fr (support)", "contact@exemple.fr", "contact", "exemple.fr"},
		{"local part case kept, domain lowered", "Jean.Dupont@Exemple.FR", "Jean.Dupont@exemple.fr", "Jean.Dupont", "exemple.fr"},
		{"plus tag kept", "jean+promo@exemple.fr", "jean+promo@exemple.fr", "jean+promo", "exemple.fr"},
		{"subdomain", "contact@mail.exemple.co.uk", "contact@mail.exemple.co.uk", "contact", "mail.exemple.co.uk"},
		{"IDN domain to A-labels", "contact@bücher.de", "contact@xn--bcher-kva.de", "contact", "xn--bcher-kva.de"},
		{"UTF-8 local part kept", "élodie@exemple.fr", "élodie@exemple.fr", "élodie", "exemple.fr"},
		{"quoted local part", `"jean dupont"@exemple.fr`, `"jean dupont"@exemple.fr`, "jean dupont", "exemple.fr"},
		{"quoted local part with @", `"a@b"@exemple.fr`, `"a@b"@exemple.fr`, "a@b", "exemple.fr"},
		{"64-octet local part", strings.Repeat("a", 64) + "@exemple.fr", strings.Repeat("a", 64) + "@exemple.fr", strings.Repeat("a", 64), "exemple.fr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseEmail(tt.in)
			if err != nil {
				t.Fatalf("ParseEmail(%q) error = %v", tt.in, err)
			}
			if got.String() != tt.wantEmail {
				t.Errorf("ParseEmail(%q).String() = %q, want %q", tt.in, got.String(), tt.wantEmail)
			}
			if got.Local() != tt.wantLocal {
				t.Errorf("ParseEmail(%q).Local() = %q, want %q", tt.in, got.Local(), tt.wantLocal)
			}
			if got.Domain().String() != tt.wantDomain {
				t.Errorf("ParseEmail(%q).Domain() = %q, want %q", tt.in, got.Domain(), tt.wantDomain)
			}
		})
	}
}

func TestParseEmailRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"no at sign", "contact.exemple.fr"},
		{"empty local part", "@exemple.fr"},
		{"empty quoted local part", `""@exemple.fr`},
		{"no domain", "contact@"},
		{"two addresses", "a@exemple.fr, b@exemple.fr"},
		{"two at signs", "a@b@exemple.fr"},
		{"final dot of the domain", "contact@exemple.fr."},
		{"dot at the end of the local part", "contact.@exemple.fr"},
		{"65-octet local part", strings.Repeat("a", 65) + "@exemple.fr"},
		{"single-label domain", "contact@localhost"},
		{"unknown TLD", "contact@exemple.invalidetld"},
		{"public suffix alone", "contact@co.uk"},
		{"URL characters in the domain", "contact@exemple.fr#autre.fr"},
		{"domain literal", "contact@[192.0.2.10]"},
		{"IPv4 domain", "contact@192.0.2.10"},
		{"URL", "https://exemple.fr"},
		{"trailing newline", "contact@exemple.fr\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseEmail(tt.in)
			if !errors.Is(err, domain.ErrInvalidEmail) {
				t.Fatalf("ParseEmail(%q) = %q, %v; want error %v", tt.in, got, err, domain.ErrInvalidEmail)
			}
		})
	}
}

// An email is personal data: a parse error must not quote it, since errors
// end up in logs.
func TestParseEmailErrorDoesNotQuoteInput(t *testing.T) {
	t.Parallel()

	for _, in := range []string{
		"jean.dupont@exemple.invalidetld",
		"jean.dupont@exemple.fr, marie.durand@exemple.fr",
		"jean.dupont@xn--a.fr",
		"jean.dupont@dupont.fr#x",
	} {
		_, err := domain.ParseEmail(in)
		if err == nil {
			t.Fatalf("ParseEmail(%q) error = nil, want an error", in)
		}
		for _, part := range []string{"jean", "dupont", "marie", "durand", "invalidetld", "xn--a"} {
			if strings.Contains(err.Error(), part) {
				t.Errorf("ParseEmail(%q) error %q quotes %q", in, err, part)
			}
		}
	}
}

func TestEmailRedacted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"jean.dupont@exemple.fr", "***@exemple.fr"},
		{"a@mail.exemple.co.uk", "***@mail.exemple.co.uk"},
		{"contact@bücher.de", "***@xn--bcher-kva.de"},
	}
	for _, tt := range tests {
		e, err := domain.ParseEmail(tt.in)
		if err != nil {
			t.Fatalf("ParseEmail(%q) error = %v", tt.in, err)
		}
		if got := e.Redacted(); got != tt.want {
			t.Errorf("ParseEmail(%q).Redacted() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func FuzzParseEmail(f *testing.F) {
	for _, s := range []string{
		"contact@exemple.fr", "Jean <Jean.Dupont@Exemple.FR>", `"a@b"@exemple.fr`, "élodie@bücher.de",
		"contact@exemple.fr#x", "contact@[192.0.2.10]", "a@b@c.fr", "@exemple.fr", "",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		e, err := domain.ParseEmail(s)
		if err != nil {
			if !errors.Is(err, domain.ErrInvalidEmail) {
				t.Fatalf("ParseEmail(%q) error %v does not match ErrInvalidEmail", s, err)
			}
			return
		}
		checkEmailInvariants(t, s, e)
	})
}

// checkEmailInvariants fails t when e, parsed from s, is not a normalised
// address.
func checkEmailInvariants(t *testing.T, s string, e domain.Email) {
	t.Helper()
	if e.Local() == "" || len(e.Local()) > 64 {
		t.Fatalf("ParseEmail(%q).Local() = %q, want 1 to 64 octets", s, e.Local())
	}
	if !strings.HasSuffix(e.String(), "@"+e.Domain().String()) {
		t.Fatalf("ParseEmail(%q) = %q, does not end with its domain %q", s, e, e.Domain())
	}
	if e.Redacted() != "***@"+e.Domain().String() {
		t.Fatalf("ParseEmail(%q).Redacted() = %q, want the domain only", s, e.Redacted())
	}
	again, err := domain.ParseEmail(e.String())
	if err != nil || again != e {
		t.Fatalf("ParseEmail(%q) = %v, %v; want %v (idempotent)", e.String(), again, err, e)
	}
}

func BenchmarkParseEmail(b *testing.B) {
	for _, in := range []string{"contact@exemple.fr", "Service client <Jean.Dupont@Mail.Exemple.co.uk>", "élodie@bücher.de"} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := domain.ParseEmail(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
