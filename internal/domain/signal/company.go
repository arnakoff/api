// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Sources of the company registers, as shown to users.
const (
	SIRENESource domain.Source = "SIRENE"
	BODACCSource domain.Source = "BODACC"
)

// companyAdvice is the advice of company_closed and company_procedure.
const companyAdvice = "Vérifie que l'entreprise exerce encore avant de lui verser de l'argent."

// noticeFamilies are the BODACC's own labels of the families read, quoted
// in company_procedure.
var noticeFamilies = map[domain.NoticeKind]string{
	domain.NoticeCollectiveProcedure:  "Procédures collectives",
	domain.NoticeRadiation:            "Radiations",
	domain.NoticeConciliation:         "Procédures de conciliation",
	domain.NoticeProfessionalRecovery: "Procédures de rétablissement professionnel",
}

// CompanyClosed returns the attention stating that the SIRENE register
// lists c's head office as closed, with the closing date when it published
// one; false when the head office is open or the record has no address.
func CompanyClosed(c domain.Company, now time.Time) (domain.Signal, bool) {
	if !c.HeadOfficeClosed || c.URL == "" {
		return domain.Signal{}, false
	}
	since := ""
	if !c.HeadOfficeClosedOn.IsZero() {
		since = " depuis le " + domain.FormatDate(c.HeadOfficeClosedOn)
	}
	message := "Le siège de " + c.Name + " est fermé" + since + " (répertoire SIRENE)."
	return companyClosed.signal(message, companyAdvice, c.URL, now), true
}

// CompanyProcedure returns the attention stating that the BODACC published a notice of one of the families read about the
// company, citing the most recent of notices; false when there is none, or
// it has an unknown family or no address.
func CompanyProcedure(notices []domain.Notice, now time.Time) (domain.Signal, bool) {
	if len(notices) == 0 {
		return domain.Signal{}, false
	}
	latest := notices[0]
	for _, n := range notices[1:] {
		if n.Published.After(latest.Published) {
			latest = n
		}
	}
	family, ok := noticeFamilies[latest.Kind]
	if !ok || latest.URL == "" {
		return domain.Signal{}, false
	}
	message := "Le BODACC a publié le " + domain.FormatDate(latest.Published) +
		" un avis de la famille « " + family + " » sur cette entreprise."
	return companyProcedure.signal(message, companyAdvice, latest.URL, now), true
}
