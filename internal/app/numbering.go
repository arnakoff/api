// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// Numbering is the numbering plan a Check reads.
type Numbering struct {
	Reader NumberingReader
}

func (Numbering) source() domain.Source { return signal.NumberingSource }

// read returns the status of the plan about in, and its signal. The plan
// covers the phone numbers of the French numbering plan only.
func (n Numbering) read(in domain.Input) (domain.SourceStatus, domain.Signal, bool) {
	p, ok := in.(domain.Phone)
	if !ok || !domain.NumberingPlanCovers(p) {
		return domain.SourceNotApplicable, domain.Signal{}, false
	}
	plan, fetchedAt, err := n.Reader.Numbering()
	if err != nil {
		return domain.SourceUnavailable, domain.Signal{}, false
	}
	s, found := signal.Numbering(p, plan, fetchedAt)
	return domain.SourceOK, s, found
}
