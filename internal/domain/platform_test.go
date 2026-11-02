// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

func TestParseRegistrable(t *testing.T) {
	t.Parallel()

	if d, err := domain.ParseRegistrable("linktr.ee"); err != nil || d.String() != "linktr.ee" {
		t.Errorf(`ParseRegistrable("linktr.ee") = %v, %v; want linktr.ee`, d, err)
	}
	for _, s := range []string{"www.linktr.ee", "LINKTR.EE", "https://linktr.ee", "linktr.ee/x", "ee", ""} {
		if _, err := domain.ParseRegistrable(s); !errors.Is(err, domain.ErrInvalidDomain) {
			t.Errorf("ParseRegistrable(%q) error = %v; want ErrInvalidDomain", s, err)
		}
	}
}

func TestNewPlatformsRejectsInvalidDomains(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"not registrable": {"sites.google.com"},
		"not normalised":  {"T.me"},
		"listed twice":    {"t.me", "x.com", "t.me"},
	}
	for name, domains := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewPlatforms(domains); !errors.Is(err, domain.ErrInvalidDomain) {
				t.Errorf("NewPlatforms(%q) error = %v; want ErrInvalidDomain", domains, err)
			}
		})
	}
}

func TestPlatformsPageStandsForSite(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPlatforms([]string{"linktr.ee", "facebook.com"})
	if err != nil {
		t.Fatalf("NewPlatforms() error = %v", err)
	}
	tests := []struct {
		host string
		want bool
	}{
		{"coinhouse.com", true},
		{"www.coinhouse.com", true},
		{"traderepublic.com", true},
		// A subdomain may belong to anyone the registrant hosts.
		{"sites.google.com", false},
		{"fr.dfi-labs.com", false},
		// A listed platform hosts its users' pages under paths.
		{"linktr.ee", false},
		{"www.facebook.com", false},
		// A private suffix of the Public Suffix List gives each user a
		// registrable domain of their own.
		{"exemple.github.io", true},
	}
	for _, tt := range tests {
		if got := p.PageStandsForSite(mustDomain(t, tt.host)); got != tt.want {
			t.Errorf("PageStandsForSite(%s) = %v; want %v", tt.host, got, tt.want)
		}
	}
	if !(domain.Platforms{}).PageStandsForSite(mustDomain(t, "linktr.ee")) {
		t.Error("empty Platforms: PageStandsForSite(linktr.ee) = false; want true")
	}
}
