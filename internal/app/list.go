// SPDX-License-Identifier: AGPL-3.0-only

package app

import "github.com/arnakoff/api/internal/domain"

// list is a published list held in memory: a Blacklist or a Whitelist.
type list interface {
	source() domain.Source
	// read returns the status of the list about in, and its signal if in
	// is listed.
	read(in domain.Input) (domain.SourceStatus, domain.Signal, bool)
}

// readLists appends to states the state of each list about in, and to
// signals the signals they give.
func readLists[L list](lists []L, in domain.Input, signals []domain.Signal, states []domain.SourceState) ([]domain.Signal, []domain.SourceState) {
	for _, l := range lists {
		status, s, ok := l.read(in)
		states = append(states, domain.SourceState{Source: l.source(), Status: status})
		if ok {
			signals = append(signals, s)
		}
	}
	return signals, states
}
