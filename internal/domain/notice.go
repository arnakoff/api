// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

// NoticeKind is the family of a BODACC notice that bears on a company's
// state. Each is a BODACC family as the BODACC names it, kept apart: what
// a family means for a signal is the signal's rule.
type NoticeKind int

// The BODACC families read; the BODACC's own label is quoted.
const (
	// NoticeCollectiveProcedure is « Procédures collectives ».
	NoticeCollectiveProcedure NoticeKind = iota + 1
	// NoticeRadiation is « Radiations ».
	NoticeRadiation
	// NoticeConciliation is « Procédures de conciliation ».
	NoticeConciliation
	// NoticeProfessionalRecovery is « Procédures de rétablissement
	// professionnel ».
	NoticeProfessionalRecovery
)

// Notice is a notice the BODACC published about a company, and nothing
// about the people it names.
type Notice struct {
	// ID is the BODACC's identifier of the notice, such as "A202601993983".
	ID string
	// Kind is the notice's family.
	Kind NoticeKind
	// Published is the day the BODACC published the notice.
	Published time.Time
}
