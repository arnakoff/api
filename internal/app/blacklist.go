// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// Blacklist is a blacklist a Check reads, and the rule its listings follow.
type Blacklist struct {
	Rule   signal.Blacklist
	Reader BlacklistReader
}

func (b Blacklist) source() domain.Source { return b.Rule.Source() }

// read returns the status of b about in, and its signal if in is listed. A
// blacklist lists sites and addresses only.
func (b Blacklist) read(in domain.Input) (domain.SourceStatus, domain.Signal, bool) {
	switch in.(type) {
	case domain.Domain, domain.Email:
	default:
		return domain.SourceNotApplicable, domain.Signal{}, false
	}
	list, fetchedAt, err := b.Reader.Blacklist()
	if err != nil {
		return domain.SourceUnavailable, domain.Signal{}, false
	}
	s, ok := b.Rule.Signal(list.Listings(in), fetchedAt)
	return domain.SourceOK, s, ok
}
