// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"slices"
	"strings"
)

// directory indexes the entries of one published list by the site, address,
// phone number or SIREN they name. Once built it is read-only: readers
// share it. Its zero value is empty and ready to fill.
type directory[T any] struct {
	sites  map[string][]T
	emails map[Email][]T
	phones map[Phone][]T
	sirens map[SIREN][]T
}

// addSite lists the whole site d: its host and every subdomain of it. A
// "www." host names its site, so www.exemple.fr lists exemple.fr, unless
// that is a public suffix.
func (dir *directory[T]) addSite(d Domain, entry T) {
	site := d.name
	if d.name != d.registrable {
		site = strings.TrimPrefix(site, "www.")
	}
	dir.sites = addTo(dir.sites, site, entry)
}

func (dir *directory[T]) addEmail(e Email, entry T) { dir.emails = addTo(dir.emails, e, entry) }

func (dir *directory[T]) addPhone(p Phone, entry T) { dir.phones = addTo(dir.phones, p, entry) }

func (dir *directory[T]) addSIREN(n SIREN, entry T) { dir.sirens = addTo(dir.sirens, n, entry) }

// addTo appends entry to the entries of k in m, making m if it is nil.
func addTo[K comparable, T any](m map[K][]T, k K, entry T) map[K][]T {
	if m == nil {
		m = make(map[K][]T)
	}
	m[k] = append(m[k], entry)
	return m
}

// lookup returns the entries that cover in, nil when none does: for a
// domain, those of its host, then of each host it is a subdomain of; for a
// SIRET, those of its SIREN; for any other input, those of the input
// itself. A listed site thus covers its subdomains, never its parent. The
// caller must not change the entries it gets.
func (dir *directory[T]) lookup(in Input) []T {
	switch v := in.(type) {
	case Domain:
		return dir.siteEntries(v.name)
	case Email:
		return dir.emails[v]
	case Phone:
		return dir.phones[v]
	case SIREN:
		return dir.sirens[v]
	case SIRET:
		return dir.sirens[v.SIREN()]
	default:
		return nil
	}
}

// siteEntries returns the entries of host and of the hosts it is a
// subdomain of. It allocates only when two of them are listed.
func (dir *directory[T]) siteEntries(host string) []T {
	var out []T
	for {
		if es := dir.sites[host]; es != nil {
			if out == nil {
				out = es
			} else {
				out = append(slices.Clip(out), es...)
			}
		}
		i := strings.IndexByte(host, '.')
		if i < 0 {
			return out
		}
		host = host[i+1:]
	}
}
