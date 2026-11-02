// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "fmt"

// Platforms are the shared platforms that host their users' pages under a
// path of their own registrable domain (linktr.ee/x, t.me/x): a page there
// is a user's, never the platform's site. Its zero value lists none.
type Platforms struct {
	registrable map[string]struct{}
}

// NewPlatforms returns the Platforms of domains, each a normalised
// registrable domain listed once.
func NewPlatforms(domains []string) (Platforms, error) {
	p := Platforms{registrable: make(map[string]struct{}, len(domains))}
	for _, s := range domains {
		if _, err := ParseRegistrable(s); err != nil {
			return Platforms{}, err
		}
		if _, dup := p.registrable[s]; dup {
			return Platforms{}, fmt.Errorf("%w: platform %q listed twice", ErrInvalidDomain, s)
		}
		p.registrable[s] = struct{}{}
	}
	return p, nil
}

// PageStandsForSite reports whether a list entry that names a page of the
// host d stands for the whole site of d. It does when d is its registrable
// domain, with or without "www.", and that domain is no shared platform: a
// subdomain may be one of many sites its registrant hosts
// (sites.google.com/view/x), and a platform hosts its users' pages.
func (p Platforms) PageStandsForSite(d Domain) bool {
	if d.name != d.registrable && d.name != "www."+d.registrable {
		return false
	}
	_, shared := p.registrable[d.registrable]
	return !shared
}
