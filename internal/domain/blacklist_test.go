// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func listing(category string, day int) domain.Listing {
	return domain.Listing{
		Categories: []string{category},
		ListedOn:   time.Date(2026, time.October, day, 0, 0, 0, 0, time.UTC),
	}
}

// testBlacklist lists a host, a subdomain of a shared host and an address.
func testBlacklist(t testing.TB) *domain.Blacklist {
	t.Helper()
	b := domain.NewBlacklist()
	b.AddSite(mustDomain(t, "armen.ltd"), listing("Usurpation", 8))
	b.AddSite(mustDomain(t, "x.hebergeur.fr"), listing("Forex", 6))
	b.AddSite(mustDomain(t, "a.x.hebergeur.fr"), listing("Crypto-actifs", 7))
	b.AddSite(mustDomain(t, "www.gcitrading.com"), listing("Forex", 1))
	b.AddSite(mustDomain(t, "www.github.io"), listing("Autres", 2))
	b.AddEmail(mustEmail(t, "contact@paris-majeste.com"), listing("Usurpation", 5))
	return b
}

func TestBlacklistListings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   domain.Input
		want []domain.Listing
	}{
		{"listed host", mustDomain(t, "armen.ltd"), []domain.Listing{listing("Usurpation", 8)}},
		{"subdomain of a listed host", mustDomain(t, "www.armen.ltd"), []domain.Listing{listing("Usurpation", 8)}},
		{"deep subdomain", mustDomain(t, "a.b.armen.ltd"), []domain.Listing{listing("Usurpation", 8)}},
		{"parent of a listed host", mustDomain(t, "hebergeur.fr"), nil},
		{"sibling of a listed host", mustDomain(t, "y.hebergeur.fr"), nil},
		{"host and parent both listed", mustDomain(t, "a.x.hebergeur.fr"), []domain.Listing{listing("Crypto-actifs", 7), listing("Forex", 6)}},
		{"look-alike suffix", mustDomain(t, "farmen.ltd"), nil},
		{"site of a www entry", mustDomain(t, "gcitrading.com"), []domain.Listing{listing("Forex", 1)}},
		{"subdomain of a www entry's site", mustDomain(t, "fr.gcitrading.com"), []domain.Listing{listing("Forex", 1)}},
		{"www entry itself", mustDomain(t, "www.gcitrading.com"), []domain.Listing{listing("Forex", 1)}},
		{"www entry on a public suffix", mustDomain(t, "www.github.io"), []domain.Listing{listing("Autres", 2)}},
		{"sibling of a www entry on a public suffix", mustDomain(t, "x.github.io"), nil},
		{"listed address", mustEmail(t, "contact@paris-majeste.com"), []domain.Listing{listing("Usurpation", 5)}},
		{"other address at the domain", mustEmail(t, "info@paris-majeste.com"), nil},
		{"address at a listed host", mustEmail(t, "info@armen.ltd"), nil},
		{"phone", mustPhone(t, "+33612345678"), nil},
	}
	b := testBlacklist(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := b.Listings(tt.in)
			if !slices.EqualFunc(got, tt.want, equalListing) {
				t.Errorf("Listings(%s) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestBlacklistListingsKeepsTheIndexIntact(t *testing.T) {
	t.Parallel()

	b := testBlacklist(t)
	in := mustDomain(t, "a.x.hebergeur.fr")
	got := b.Listings(in)
	got[0] = domain.Listing{}

	if again := b.Listings(in); !equalListing(again[0], listing("Crypto-actifs", 7)) {
		t.Errorf("Listings after a change by the caller = %v, want the index untouched", again)
	}
	if parent := b.Listings(mustDomain(t, "x.hebergeur.fr")); len(parent) != 1 {
		t.Errorf("Listings(x.hebergeur.fr) = %v, want its one listing", parent)
	}
}

func mustDomain(t testing.TB, s string) domain.Domain { return must(t, s, domain.ParseDomain) }

func mustEmail(t testing.TB, s string) domain.Email { return must(t, s, domain.ParseEmail) }

func mustPhone(t testing.TB, s string) domain.Phone { return must(t, s, domain.ParsePhone) }

func must[T any](t testing.TB, s string, parse func(string) (T, error)) T {
	t.Helper()
	v, err := parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}

func equalListing(a, b domain.Listing) bool {
	return slices.Equal(a.Categories, b.Categories) && a.ListedOn.Equal(b.ListedOn)
}

func BenchmarkBlacklistListings(b *testing.B) {
	list := testBlacklist(b)
	for _, in := range []domain.Input{mustDomain(b, "www.absent.example.co.uk"), mustEmail(b, "absent@exemple.fr"), mustDomain(b, "www.armen.ltd")} {
		b.Run(in.String(), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				list.Listings(in)
			}
		})
	}
}
