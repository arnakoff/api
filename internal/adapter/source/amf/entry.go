// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"strings"

	"github.com/arnakoff/api/internal/domain"
)

// Entry kinds written in the kind column, the names of domain.Kind.
var (
	kindDomain = domain.KindDomain.String()
	kindEmail  = domain.KindEmail.String()
)

// classify reads one "nom" of the list. An address gives kind "email" and
// the normalised address as key; a site gives kind "domain", the normalised
// host as key and, when the entry targets a page rather than the whole
// site, its path and query. Anything else, a free-text name that may name a
// natural person, is not ok: it is left out of the snapshot.
func classify(nom string) (kind, key, path string, ok bool) {
	nom = strings.TrimSpace(nom)
	if nom == "" || strings.ContainsAny(nom, " \t") {
		return "", "", "", false
	}
	if strings.Contains(nom, "@") {
		// A final dot, as in "prénom.nom@lgt-bp.com.", is a typo of the list.
		e, err := domain.ParseEmail(strings.TrimSuffix(nom, "."))
		if err != nil {
			return "", "", "", false
		}
		return kindEmail, e.String(), "", true
	}
	d, err := domain.ParseDomain(nom)
	if err != nil {
		return "", "", "", false
	}
	return kindDomain, d.String(), pathOf(nom), true
}

// pathOf returns what follows the host in a site entry, or "" when that is
// only a slash or an empty query: the entry then targets the whole site.
func pathOf(site string) string {
	if _, rest, found := strings.Cut(site, "://"); found {
		site = rest
	}
	i := strings.IndexAny(site, "/?#")
	if i < 0 || strings.Trim(site[i:], "/?#") == "" {
		return ""
	}
	return site[i:]
}
