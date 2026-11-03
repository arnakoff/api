// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Definition is a signal a check can give, as GET /v1/signals and
// docs/signals.md list it. A code given by several sources has one
// definition per source, as a signal names one source.
type Definition struct {
	// Code is the stable identifier of the rule, Signal.Code.
	Code string
	// Level is the level of the signal.
	Level domain.Level
	// Inputs are the kinds of input that can give the signal.
	Inputs []domain.Kind
	// Source is the source the signal cites.
	Source domain.Source
	// Condition says when the signal is given, in French.
	Condition string
}

// Kinds of input the rules read.
var (
	webInputs     = []domain.Kind{domain.KindDomain, domain.KindEmail}
	companyInputs = []domain.Kind{domain.KindSIREN, domain.KindSIRET}
	listedInputs  = []domain.Kind{domain.KindDomain, domain.KindEmail, domain.KindPhone, domain.KindSIREN, domain.KindSIRET}
	phoneInputs   = []domain.Kind{domain.KindPhone}
	ibanInputs    = []domain.Kind{domain.KindIBAN}
)

// Conditions shared by several definitions.
const (
	blacklistedCondition = "Site inscrit ou adresse e-mail inscrite à l'identique"
	registeredCondition  = "Site, adresse, numéro ou SIREN d'une autorisation en cours"
)

// The definitions the rules give their signals from, in document order.
var (
	amfBlacklisted  = Definition{"amf_blacklist", domain.LevelAlert, webInputs, AMFSource, blacklistedCondition}
	acprBlacklisted = Definition{"acpr_blacklist", domain.LevelAlert, webInputs, ACPRSource, blacklistedCondition}
	psanRegistered  = Definition{"amf_registered", domain.LevelInfo, listedInputs, PSANSource, registeredCondition}
	offerRegistered = Definition{"amf_registered", domain.LevelInfo, listedInputs, OffersSource, registeredCondition}
	psanEnded       = Definition{
		"amf_registration_ended", domain.LevelInfo, listedInputs, PSANSource,
		"Toutes les autorisations trouvées sont radiées",
	}
	domainRecent = Definition{
		"domain_recent", domain.LevelAttention, webInputs, RDAPSource, "Domaine créé il y a moins de 30 jours",
	}
	domainLookalike = Definition{
		"domain_lookalike", domain.LevelAttention, webInputs, LookalikeSource,
		"Domaine qui ressemble à un domaine officiel de la liste de référence",
	}
	phoneOperator   = Definition{"phone_operator", domain.LevelInfo, phoneInputs, NumberingSource, "Numéro dans une tranche attribuée"}
	phoneUnassigned = Definition{
		"phone_unassigned", domain.LevelAttention, phoneInputs, NumberingSource,
		"Numéro du plan français hors de toute tranche attribuée",
	}
	companyClosed    = Definition{"company_closed", domain.LevelAttention, companyInputs, SIRENESource, "Établissement siège fermé"}
	companyProcedure = Definition{
		"company_procedure", domain.LevelAttention, companyInputs, BODACCSource,
		"Avis publié de procédure collective, radiation, conciliation ou rétablissement professionnel",
	}
	ibanCountry = Definition{
		"iban_country", domain.LevelInfo, ibanInputs, IBANSource, "Pays de tenue du compte, d'après le code pays de l'IBAN",
	}
)

// catalog lists every definition, in document order.
var catalog = []Definition{
	amfBlacklisted, acprBlacklisted, psanRegistered, offerRegistered, psanEnded, domainRecent, domainLookalike,
	phoneOperator, phoneUnassigned, companyClosed, companyProcedure, ibanCountry,
}

// Catalog returns every signal a check can give, in document order. The
// result is the caller's to change.
func Catalog() []Definition {
	out := slices.Clone(catalog)
	for i := range out {
		out[i].Inputs = slices.Clone(out[i].Inputs)
	}
	return out
}

// InputNames returns the names of d's inputs, as Kind.String writes them
// and the contract's InputType lists them.
func (d Definition) InputNames() []string {
	names := make([]string, 0, len(d.Inputs))
	for _, k := range d.Inputs {
		names = append(names, k.String())
	}
	return names
}

// signal returns the signal of d stating message, as visible writes it,
// with advice, citing url and observed at observedAt.
func (d Definition) signal(message, advice, url string, observedAt time.Time) domain.Signal {
	return domain.Signal{
		Code: d.Code, Level: d.Level, Message: visible(message), Advice: advice,
		Source: d.Source, SourceURL: url, ObservedAt: observedAt,
	}
}
