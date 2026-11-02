// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"strconv"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// RDAPSource names the domain registries, read over RDAP, as shown to
// users.
const RDAPSource domain.Source = "RDAP"

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
// record the registry published; false when it is older, or
// the record has no registration date, a date after now or no address.
// Dates are shown in now's location.
func DomainRecent(d domain.Domain, reg domain.Registration, now time.Time) (domain.Signal, bool) {
	age := now.Sub(reg.Registered)
	if reg.Registered.IsZero() || reg.URL == "" || age < 0 || age >= RecentDomainAge {
		return domain.Signal{}, false
	}
	return recentSignal(d, reg, age, now), true
}

// recentSignal returns domain_recent for d, registered age before now. It
// is kept out of DomainRecent, which runs in a goroutine of its own on every
// check: built there, it slowed BenchmarkCheck/contact@exemple.fr by 40 %.
func recentSignal(d domain.Domain, reg domain.Registration, age time.Duration, now time.Time) domain.Signal {
	message := "Le domaine " + d.Registrable() + " a été créé le " + domain.FormatDate(reg.Registered.In(now.Location())) +
		", il y a " + daysAgo(age) + " (registre RDAP)."
	return domainRecent.signal(message, recentAdvice, reg.URL, now)
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
