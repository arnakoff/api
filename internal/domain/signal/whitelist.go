// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"slices"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Sources of the AMF white lists, as shown to users.
const (
	PSANSource   domain.Source = "AMF PSAN"
	OffersSource domain.Source = "AMF biens divers"
)

// whitelistsURL is the AMF page of its white lists, checked on 30/10/2026.
const whitelistsURL = "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-blanches"

// impersonation is the advice of a registered actor: scammers impersonate
// them (the « Usurpation » category of the blacklists).
const impersonation = "Des sites imitent des acteurs enregistrés : vérifie l'adresse exacte sur le site de l'AMF avant de verser de l'argent."

// Whitelist is a published white list and the signals its authorisations
// give.
type Whitelist struct {
	source domain.Source
	list   string // the list, as the message names it
	label  func(domain.Authorisation) string
}

// AMFPSANWhitelist is the AMF list of digital asset service providers,
// labelled by their kind of authorisation.
func AMFPSANWhitelist() Whitelist {
	return Whitelist{
		source: PSANSource,
		list:   "la liste des prestataires sur actifs numériques de l'AMF",
		label:  func(a domain.Authorisation) string { return a.Kind },
	}
}

// AMFOffersWhitelist is the AMF list of registered offers in goods,
// labelled by their number.
func AMFOffersWhitelist() Whitelist {
	return Whitelist{
		source: OffersSource,
		list:   "la liste des offres de biens divers enregistrées par l'AMF",
		label:  func(a domain.Authorisation) string { return "n° " + a.Number },
	}
}

// Source returns the source of the signals.
func (w Whitelist) Source() domain.Source { return w.source }

// Signal returns the info stating that the checked input is on the white
// list, from its authorisations there and the time the list was fetched;
// false when it has none. Running authorisations give amf_registered, since
// the first of them started; when all have ended, they give
// amf_registration_ended, from the first start to the last end.
func (w Whitelist) Signal(auths []domain.Authorisation, observedAt time.Time) (domain.Signal, bool) {
	if len(auths) == 0 {
		return domain.Signal{}, false
	}
	s := domain.Signal{Level: domain.LevelInfo, Source: w.source, SourceURL: whitelistsURL, ObservedAt: observedAt}
	running := slices.DeleteFunc(slices.Clone(auths), domain.Authorisation.Ended)
	if len(running) > 0 {
		s.Code, s.Advice = "amf_registered", impersonation
		s.Message = "Figure sur " + w.list + " : " + w.names(running) + " depuis le " + domain.FormatDate(firstStart(running)) + "."
		return s, true
	}
	s.Code = "amf_registration_ended"
	s.Message = "A figuré sur " + w.list + " : " + w.names(auths) + " du " +
		domain.FormatDate(firstStart(auths)) + " au " + domain.FormatDate(lastEnd(auths)) + "."
	return s, true
}

// names returns « NAME (label) » for each authorisation, each once, in
// order.
func (w Whitelist) names(auths []domain.Authorisation) string {
	var out []string
	for _, a := range auths {
		if n := a.Name + " (" + w.label(a) + ")"; !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return strings.Join(out, ", ")
}

// firstStart returns the earliest start of auths, which are not empty.
func firstStart(auths []domain.Authorisation) time.Time {
	return slices.MinFunc(auths, func(a, b domain.Authorisation) int { return a.From.Compare(b.From) }).From
}

// lastEnd returns the latest end of auths, which are not empty.
func lastEnd(auths []domain.Authorisation) time.Time {
	return slices.MaxFunc(auths, func(a, b domain.Authorisation) int { return a.Until.Compare(b.Until) }).Until
}
