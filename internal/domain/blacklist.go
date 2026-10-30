// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

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
	dir directory[Listing]
}

// NewBlacklist returns an empty Blacklist.
func NewBlacklist() *Blacklist {
	return &Blacklist{}
}

// AddSite lists the whole site d: its host and every subdomain of it. A
// "www." host names its site, so www.exemple.fr lists exemple.fr, unless
// that is a public suffix.
func (b *Blacklist) AddSite(d Domain, l Listing) { b.dir.addSite(d, l) }

// AddEmail lists the address e alone.
func (b *Blacklist) AddEmail(e Email, l Listing) { b.dir.addEmail(e, l) }

// Listings returns the listings that cover in, nil when none does: for an
// address, those of the address itself; for a domain, those of its host,
// then of each host it is a subdomain of. A listed site thus covers its
// subdomains, never its parent. The caller must not change the listings it
// gets.
func (b *Blacklist) Listings(in Input) []Listing { return b.dir.lookup(in) }
