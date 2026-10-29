// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

func TestParseDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		in              string
		wantName        string
		wantRegistrable string
	}{
		// Entries as published in the AMF blacklist on 29/10/2026.
		{"AMF entry with https scheme", "https://www.groupe-ytria.com", "www.groupe-ytria.com", "groupe-ytria.com"},
		{"AMF entry with http scheme", "http://audubert-conseil.com", "audubert-conseil.com", "audubert-conseil.com"},
		{"AMF entry with upper case", "www.BourseBinaire.fr", "www.boursebinaire.fr", "boursebinaire.fr"},
		{"AMF entry with path", "investpathline.com/fr/3/", "investpathline.com", "investpathline.com"},
		{"AMF entry with empty query", "optimalduck.autos/?", "optimalduck.autos", "optimalduck.autos"},
		{"AMF entry with query", "www.uniglobemarkets.com/?lang=fr", "www.uniglobemarkets.com", "uniglobemarkets.com"},
		{"AMF entry under a two-label suffix", "www.atos-limited.co.uk", "www.atos-limited.co.uk", "atos-limited.co.uk"},
		{"AMF entry under a new generic TLD", "armen.ltd", "armen.ltd", "armen.ltd"},

		{"surrounding spaces", "  exemple.fr \t", "exemple.fr", "exemple.fr"},
		{"final dot", "exemple.fr.", "exemple.fr", "exemple.fr"},
		{"upper-case scheme", "HTTPS://EXEMPLE.FR/", "exemple.fr", "exemple.fr"},
		{"port", "https://exemple.fr:8443/login", "exemple.fr", "exemple.fr"},
		{"fragment", "exemple.fr#contact", "exemple.fr", "exemple.fr"},
		{"credentials before the host are ignored", "https://amf-france.org@exemple.com", "exemple.com", "exemple.com"},
		{"credentials without a scheme are ignored", "amf-france.org:x@exemple.com/", "exemple.com", "exemple.com"},
		{"backslash ends the host as in browsers", `https://exemple.com\@amf-france.org`, "exemple.com", "exemple.com"},
		{"scheme-like text after the host", "exemple.fr/r?u=https://autre.fr", "exemple.fr", "exemple.fr"},
		{"IDN to A-labels", "bücher.de", "xn--bcher-kva.de", "xn--bcher-kva.de"},
		{"IDN upper case and full width", "ＢÜCHER.de", "xn--bcher-kva.de", "xn--bcher-kva.de"},
		{"A-labels kept", "xn--bcher-kva.de", "xn--bcher-kva.de", "xn--bcher-kva.de"},
		{"private suffix", "arnaque.github.io", "arnaque.github.io", "arnaque.github.io"},
		{"63-character label", strings.Repeat("a", 63) + ".fr", strings.Repeat("a", 63) + ".fr", strings.Repeat("a", 63) + ".fr"},
		{"253-character name", name253, name253, "aa.fr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseDomain(tt.in)
			if err != nil {
				t.Fatalf("ParseDomain(%q) error = %v", tt.in, err)
			}
			if got.String() != tt.wantName {
				t.Errorf("ParseDomain(%q).String() = %q, want %q", tt.in, got.String(), tt.wantName)
			}
			if got.Registrable() != tt.wantRegistrable {
				t.Errorf("ParseDomain(%q).Registrable() = %q, want %q", tt.in, got.Registrable(), tt.wantRegistrable)
			}
		})
	}
}

// name253 is a 253-character name, the longest RFC 1035 allows: 124 "a."
// labels and "aa.fr" (248 + 5 characters).
var name253 = strings.Repeat("a.", 124) + "aa.fr"

func TestParseDomainRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"empty", "", domain.ErrInvalidDomain},
		{"spaces only", "   ", domain.ErrInvalidDomain},
		{"scheme without host", "https://", domain.ErrInvalidDomain},
		{"ftp scheme", "ftp://exemple.fr", domain.ErrUnsupportedScheme},
		{"javascript scheme", "javascript://exemple.fr/%0aalert(1)", domain.ErrUnsupportedScheme},
		{"opaque javascript URL", "javascript:alert(1)", domain.ErrInvalidDomain},
		{"single label", "localhost", domain.ErrInvalidDomain},
		{"public suffix alone", "co.uk", domain.ErrInvalidDomain},
		{"unknown TLD", "exemple.invalidetld", domain.ErrInvalidDomain},
		{"leading dot", ".exemple.fr", domain.ErrInvalidDomain},
		{"empty label", "exemple..fr", domain.ErrInvalidDomain},
		{"underscore", "ex_emple.fr", domain.ErrInvalidDomain},
		{"space inside", "exe mple.fr", domain.ErrInvalidDomain},
		{"leading hyphen", "-exemple.fr", domain.ErrInvalidDomain},
		{"64-character label", strings.Repeat("a", 64) + ".fr", domain.ErrInvalidDomain},
		{"254-character name", "a" + name253, domain.ErrInvalidDomain},
		{"invalid punycode", "xn--a.fr", domain.ErrInvalidDomain},
		{"IPv4", "192.0.2.10", domain.ErrIPAddress},
		{"IPv4 URL", "http://192.0.2.10/login", domain.ErrIPAddress},
		{"IPv6 URL", "https://[2001:db8::1]:8443/", domain.ErrIPAddress},
		{"decimal IPv4", "http://3221225994/", domain.ErrIPAddress},
		{"hexadecimal last label", "http://192.0.2.0x0a/", domain.ErrIPAddress},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseDomain(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseDomain(%q) = %q, %v; want error %v", tt.in, got, err, tt.wantErr)
			}
			if !errors.Is(err, domain.ErrInvalidDomain) {
				t.Errorf("ParseDomain(%q) error %v does not match ErrInvalidDomain", tt.in, err)
			}
		})
	}
}

func FuzzParseDomain(f *testing.F) {
	for _, s := range []string{
		"https://www.groupe-ytria.com", "investpathline.com/fr/3/", "www.BourseBinaire.fr",
		"https://amf-france.org@exemple.com", `https://exemple.com\@x.fr`, "bücher.de",
		"192.0.2.10", "[2001:db8::1]", "ftp://x.fr", "xn--a.fr", "a..fr", "",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		d, err := domain.ParseDomain(s)
		if err != nil {
			if !errors.Is(err, domain.ErrInvalidDomain) {
				t.Fatalf("ParseDomain(%q) error %v does not match ErrInvalidDomain", s, err)
			}
			return
		}
		name := d.String()
		if len(name) > 253 || name != strings.ToLower(name) || strings.ContainsAny(name, "/:@\\ ") {
			t.Fatalf("ParseDomain(%q) = %q, not a normalised name", s, name)
		}
		if !strings.HasSuffix(name, d.Registrable()) {
			t.Fatalf("ParseDomain(%q): registrable %q is not a suffix of %q", s, d.Registrable(), name)
		}
		again, err := domain.ParseDomain(name)
		if err != nil || again != d {
			t.Fatalf("ParseDomain(%q) = %v, %v; want %v (idempotent)", name, again, err, d)
		}
	})
}

func BenchmarkParseDomain(b *testing.B) {
	for _, in := range []string{"www.atos-limited.co.uk", "https://www.uniglobemarkets.com/?lang=fr", "bücher.de"} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := domain.ParseDomain(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
