// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// Whitelist is a white list a Check reads, and the rule its authorisations
// follow.
type Whitelist struct {
	Rule   signal.Whitelist
	Reader WhitelistReader
}

func (w Whitelist) source() domain.Source { return w.Rule.Source() }

// read returns the status of w about in, and its signal if in is listed. A
// white list lists sites, addresses, phone numbers and companies.
func (w Whitelist) read(in domain.Input) (domain.SourceStatus, domain.Signal, bool) {
	switch in.(type) {
	case domain.Domain, domain.Email, domain.Phone, domain.SIREN, domain.SIRET:
	default:
		return domain.SourceNotApplicable, domain.Signal{}, false
	}
	list, fetchedAt, err := w.Reader.Whitelist()
	if err != nil {
		return domain.SourceUnavailable, domain.Signal{}, false
	}
	s, ok := w.Rule.Signal(list.Authorisations(in), fetchedAt)
	return domain.SourceOK, s, ok
}
