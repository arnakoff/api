// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// NumberingSource is the name of the ARCEP numbering plan, as shown to
// users.
const NumberingSource domain.Source = "ARCEP"

// numberingURL is the data.gouv.fr page of the ARCEP numbering plan, which
// links to its files; checked on 02/11/2026 (decision of the owner).
const numberingURL = "https://www.data.gouv.fr/datasets/ressources-en-numerotation-telephonique"

// spoofingAdvice is the advice of the phone signals: the number a call or
// a message shows can be spoofed.
const spoofingAdvice = "Un numéro affiché peut être usurpé : rappelle l'organisme au numéro que tu trouves toi-même sur son site officiel."

// Numbering returns what the numbering plan, fetched at observedAt, says
// of p: the info phone_operator naming the holder of the range p falls in,
// or the attention phone_unassigned when p is in no assigned range. It
// returns false for a number outside the French numbering plan, which the
// plan does not cover.
func Numbering(p domain.Phone, plan *domain.NumberingPlan, observedAt time.Time) (domain.Signal, bool) {
	if !domain.NumberingPlanCovers(p) {
		return domain.Signal{}, false
	}
	s := domain.Signal{Advice: spoofingAdvice, Source: NumberingSource, SourceURL: numberingURL, ObservedAt: observedAt}
	a, ok := plan.Lookup(p)
	if !ok {
		s.Code, s.Level = "phone_unassigned", domain.LevelAttention
		s.Message = "Le numéro n'appartient à aucune tranche attribuée par l'ARCEP : il ne peut pas être en service."
		return s, true
	}
	s.Code, s.Level = "phone_operator", domain.LevelInfo
	s.Message = "Le numéro appartient à une tranche attribuée à " + holder(a) + " le " + domain.FormatDate(a.AssignedOn) +
		" (numérotation ARCEP). Le numéro a pu être porté chez un autre opérateur."
	return s, true
}

// holder names the holder of a range, « Name (CODE) », or its code alone
// when the operators file does not name it.
func holder(a domain.NumberAssignment) string {
	if a.OperatorName == "" {
		return a.Operator
	}
	return a.OperatorName + " (" + a.Operator + ")"
}
