// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"strings"

	"github.com/arnakoff/api/internal/domain"
)

// lookupKey is what an entry is looked up by: the first four columns of
// every snapshot row.
type lookupKey struct {
	kind  string // a domain.Kind name: "domain", "email", "phone" or "siren"
	value string // normalised by the domain value object
	path  string // path and query of a site entry that targets one page
	entry string // the value as the AMF lists it
}

// fields returns the key as the first columns of a row, followed by rest.
func (k lookupKey) fields(rest ...string) []string {
	row := make([]string, 0, 4+len(rest))
	row = append(row, k.kind, k.value, k.path, k.entry)
	return append(row, rest...)
}

// classify reads one "nom" of the blacklist: an address, a site, or a
// free-text name, which may name a natural person and is not ok.
func classify(nom string) (lookupKey, bool) {
	nom = strings.TrimSpace(nom)
	if nom == "" || strings.ContainsAny(nom, " \t") {
		return lookupKey{}, false
	}
	if strings.Contains(nom, "@") {
		return emailKey(nom)
	}
	return siteKey(nom)
}

// siteKey reads a site or web URL; the key is its normalised host. A
// missing "h", as in "ttps://www.alphacap.fr/", is a typo of the lists.
func siteKey(s string) (lookupKey, bool) {
	entry := strings.TrimSpace(s)
	s = entry
	if strings.HasPrefix(s, "ttp://") || strings.HasPrefix(s, "ttps://") {
		s = "h" + s
	}
	d, err := domain.ParseDomain(s)
	if err != nil {
		return lookupKey{}, false
	}
	return lookupKey{kind: domain.KindDomain.String(), value: d.String(), path: pathOf(s), entry: entry}, true
}

// emailKey reads an address. A final dot, as in "prénom.nom@lgt-bp.com.",
// is a typo of the lists.
func emailKey(s string) (lookupKey, bool) {
	s = strings.TrimSpace(s)
	e, err := domain.ParseEmail(strings.TrimSuffix(s, "."))
	if err != nil {
		return lookupKey{}, false
	}
	return lookupKey{kind: domain.KindEmail.String(), value: e.String(), entry: s}, true
}

// phoneKey reads a phone number; the key is its E.164 form.
func phoneKey(s string) (lookupKey, bool) {
	s = strings.TrimSpace(s)
	p, err := domain.ParsePhone(s)
	if err != nil {
		return lookupKey{}, false
	}
	return lookupKey{kind: domain.KindPhone.String(), value: p.String(), entry: s}, true
}

// sirenKey reads a SIREN.
func sirenKey(s string) (lookupKey, bool) {
	s = strings.TrimSpace(s)
	n, err := domain.ParseSIREN(s)
	if err != nil {
		return lookupKey{}, false
	}
	return lookupKey{kind: domain.KindSIREN.String(), value: n.String(), entry: s}, true
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
