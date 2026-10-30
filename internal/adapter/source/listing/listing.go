// SPDX-License-Identifier: AGPL-3.0-only

package listing

import (
	"strings"

	"github.com/arnakoff/api/internal/domain"
)

// Columns every snapshot row of a listing starts with.
const (
	// ColumnKind is the kind of the key: "domain", "email", "phone" or
	// "siren".
	ColumnKind = "kind"
	// ColumnKey is the normalised host, address, number or SIREN the entry
	// is looked up by.
	ColumnKey = "key"
	// ColumnPath is the path and query of a site entry that targets one
	// page, "" when it targets the whole site or the key is not a site.
	ColumnPath = "path"
	// ColumnEntry is the value as the publisher lists it.
	ColumnEntry = "entry"
)

// Columns a blacklist row adds to the common ones, for every blacklist.
const (
	// ColumnCategories are the publisher's categories, separated by
	// CategorySeparator.
	ColumnCategories = "categories"
	// ColumnListedOn is the listing date, AAAA-MM-JJ.
	ColumnListedOn = "listed_on"
)

// CategorySeparator separates the categories of a blacklist row.
const CategorySeparator = "|"

// BlacklistColumns returns the columns of every blacklist snapshot.
func BlacklistColumns() []string {
	return Columns(ColumnCategories, ColumnListedOn)
}

// Columns returns the common columns followed by rest, the source's own.
func Columns(rest ...string) []string {
	return append([]string{ColumnKind, ColumnKey, ColumnPath, ColumnEntry}, rest...)
}

// Key is what an entry is looked up by: the first four columns of every
// snapshot row.
type Key struct {
	Kind  string // a domain.Kind name: "domain", "email", "phone" or "siren"
	Value string // normalised by the domain value object
	Path  string // path and query of a site entry that targets one page
	Entry string // the value as the publisher lists it
}

// Fields returns the key as the first columns of a row, followed by rest.
func (k Key) Fields(rest ...string) []string {
	row := make([]string, 0, 4+len(rest))
	row = append(row, k.Kind, k.Value, k.Path, k.Entry)
	return append(row, rest...)
}

// Classify reads one entry of a blacklist: an address, a site, or a
// free-text name, which may name a natural person and is not ok.
func Classify(nom string) (Key, bool) {
	nom = strings.TrimSpace(nom)
	if nom == "" || strings.ContainsAny(nom, " \t") {
		return Key{}, false
	}
	if strings.Contains(nom, "@") {
		return Email(nom)
	}
	return Site(nom)
}

// Site reads a site or web URL; the key is its normalised host. A
// missing "h", as in "ttps://www.alphacap.fr/", is a typo of the lists.
func Site(s string) (Key, bool) {
	entry := strings.TrimSpace(s)
	s = entry
	if strings.HasPrefix(s, "ttp://") || strings.HasPrefix(s, "ttps://") {
		s = "h" + s
	}
	d, err := domain.ParseDomain(s)
	if err != nil {
		return Key{}, false
	}
	return Key{Kind: domain.KindDomain.String(), Value: d.String(), Path: pathOf(s), Entry: entry}, true
}

// Email reads an address. A final dot, as in "prénom.nom@lgt-bp.com.",
// is a typo of the lists.
func Email(s string) (Key, bool) {
	s = strings.TrimSpace(s)
	e, err := domain.ParseEmail(strings.TrimSuffix(s, "."))
	if err != nil {
		return Key{}, false
	}
	return Key{Kind: domain.KindEmail.String(), Value: e.String(), Entry: s}, true
}

// Phone reads a phone number; the key is its E.164 form.
func Phone(s string) (Key, bool) {
	s = strings.TrimSpace(s)
	p, err := domain.ParsePhone(s)
	if err != nil {
		return Key{}, false
	}
	return Key{Kind: domain.KindPhone.String(), Value: p.String(), Entry: s}, true
}

// SIREN reads a SIREN.
func SIREN(s string) (Key, bool) {
	s = strings.TrimSpace(s)
	n, err := domain.ParseSIREN(s)
	if err != nil {
		return Key{}, false
	}
	return Key{Kind: domain.KindSIREN.String(), Value: n.String(), Entry: s}, true
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
