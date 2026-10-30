// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"slices"
	"strings"
	"time"
)

// Listing is one inscription of a site or an address on a blacklist.
type Listing struct {
	// Categories are the publisher's categories, as it labels them.
	Categories []string
	// ListedOn is the day the publisher listed the entry.
	ListedOn time.Time
}

// Blacklist indexes the listings of one blacklist by the host or address
// they name. Once built it is read-only: readers share it.
type Blacklist struct {
	sites  map[string][]Listing
	emails map[Email][]Listing
}

// NewBlacklist returns an empty Blacklist.
func NewBlacklist() *Blacklist {
	return &Blacklist{sites: map[string][]Listing{}, emails: map[Email][]Listing{}}
}

// AddSite lists the whole site d: its host and every subdomain of it. A
// "www." host names its site, so www.exemple.fr lists exemple.fr, unless
// that is a public suffix.
func (b *Blacklist) AddSite(d Domain, l Listing) {
	site := d.name
	if d.name != d.registrable {
		site = strings.TrimPrefix(site, "www.")
	}
	b.sites[site] = append(b.sites[site], l)
}

// AddEmail lists the address e alone.
func (b *Blacklist) AddEmail(e Email, l Listing) {
	b.emails[e] = append(b.emails[e], l)
}

// Listings returns the listings that cover in, nil when none does: for an
// address, those of the address itself; for a domain, those of its host,
// then of each host it is a subdomain of. A listed site thus covers its
// subdomains, never its parent. The caller must not change the listings it
// gets.
func (b *Blacklist) Listings(in Input) []Listing {
	switch v := in.(type) {
	case Domain:
		return b.siteListings(v.name)
	case Email:
		return b.emails[v]
	default:
		return nil
	}
}

// siteListings returns the listings of host and of the hosts it is a
// subdomain of. It allocates only when two of them are listed.
func (b *Blacklist) siteListings(host string) []Listing {
	var out []Listing
	for {
		if ls := b.sites[host]; ls != nil {
			if out == nil {
				out = ls
			} else {
				out = append(slices.Clip(out), ls...)
			}
		}
		i := strings.IndexByte(host, '.')
		if i < 0 {
			return out
		}
		host = host[i+1:]
	}
}
