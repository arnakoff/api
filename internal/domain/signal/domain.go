// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"strconv"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// RecentDomainAge is the age under which a domain is recent: 30 days, the
// threshold of issue #43 (docs/signals.md). A domain exactly that old is no
// longer recent.
const RecentDomainAge = 30 * 24 * time.Hour

// day is the unit of the age shown to users.
const day = 24 * time.Hour

// recentAdvice is the advice of domain_recent.
const recentAdvice = "Un site créé il y a quelques jours n'a pas d'historique : vérifie qui l'exploite avant de verser de l'argent."

// DomainRecent returns the attention stating that d's registrable domain was
// registered less than RecentDomainAge before now, according to reg, the
// record the registry named source published; false when it is older, or
// the record has no registration date, a date after now or no address.
// Dates are shown in now's location.
func DomainRecent(d domain.Domain, reg domain.Registration, source domain.Source, now time.Time) (domain.Signal, bool) {
	age := now.Sub(reg.Registered)
	if reg.Registered.IsZero() || reg.URL == "" || age < 0 || age >= RecentDomainAge {
		return domain.Signal{}, false
	}
	return domain.Signal{
		Code:  "domain_recent",
		Level: domain.LevelAttention,
		Message: "Le domaine " + d.Registrable() + " a été créé le " + domain.FormatDate(reg.Registered.In(now.Location())) +
			", il y a " + daysAgo(age) + " (registre RDAP).",
		Advice:     recentAdvice,
		Source:     source,
		SourceURL:  reg.URL,
		ObservedAt: now,
	}, true
}

// daysAgo returns age in whole days, in French.
func daysAgo(age time.Duration) string {
	switch days := int(age / day); days {
	case 0:
		return "moins d'un jour"
	case 1:
		return "1 jour"
	default:
		return strconv.Itoa(days) + " jours"
	}
}
