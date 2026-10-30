// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"slices"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Sources of the blacklists, as shown to users.
const (
	AMFSource  domain.Source = "AMF"
	ACPRSource domain.Source = "ACPR"
)

// Blacklist is a published blacklist and the signal its listings give.
type Blacklist struct {
	code      string
	source    domain.Source
	publisher string // as the message names it
	url       string
}

// AMFBlacklist is the blacklist of the Autorité des marchés financiers.
func AMFBlacklist() Blacklist {
	return Blacklist{
		code:      "amf_blacklist",
		source:    AMFSource,
		publisher: "l'AMF",
		url:       "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde",
	}
}

// ACPRBlacklist is the blacklist the ACPR publishes with the AMF on
// Assurance Banque Épargne Info Service.
func ACPRBlacklist() Blacklist {
	return Blacklist{
		code:      "acpr_blacklist",
		source:    ACPRSource,
		publisher: "l'ACPR",
		url:       "https://www.abe-infoservice.fr/liste-noire",
	}
}

// Source returns the source of the signal.
func (b Blacklist) Source() domain.Source { return b.source }

// Signal returns the alerte stating that the checked input is on the
// blacklist, from its listings there and the time the list was fetched;
// false when it has no listing. Several listings give one signal: every
// category once, and the earliest date.
func (b Blacklist) Signal(listings []domain.Listing, observedAt time.Time) (domain.Signal, bool) {
	if len(listings) == 0 {
		return domain.Signal{}, false
	}
	return domain.Signal{
		Code:       b.code,
		Level:      domain.LevelAlert,
		Message:    b.message(categoriesOf(listings), earliest(listings)),
		Source:     b.source,
		SourceURL:  b.url,
		ObservedAt: observedAt,
	}, true
}

// message is « Figure sur la liste noire de l'AMF (catégorie Forex) depuis
// le 08/10/2026. », without the parenthesis when no category is given.
func (b Blacklist) message(categories []string, since time.Time) string {
	var m strings.Builder
	m.WriteString("Figure sur la liste noire de ")
	m.WriteString(b.publisher)
	switch len(categories) {
	case 0:
	case 1:
		m.WriteString(" (catégorie " + categories[0] + ")")
	default:
		m.WriteString(" (catégories " + strings.Join(categories, ", ") + ")")
	}
	m.WriteString(" depuis le " + domain.FormatDate(since) + ".")
	return m.String()
}

// categoriesOf returns the categories of listings, each once, in order.
// The AMF writes spaces as underscores: they read as spaces.
func categoriesOf(listings []domain.Listing) []string {
	var out []string
	for _, l := range listings {
		for _, c := range l.Categories {
			c = strings.ReplaceAll(c, "_", " ")
			if !slices.Contains(out, c) {
				out = append(out, c)
			}
		}
	}
	return out
}

// earliest returns the earliest listing date of listings, which are not
// empty.
func earliest(listings []domain.Listing) time.Time {
	first := listings[0].ListedOn
	for _, l := range listings[1:] {
		if l.ListedOn.Before(first) {
			first = l.ListedOn
		}
	}
	return first
}
