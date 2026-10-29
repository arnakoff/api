// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

// Signal is one sourced fact about a checked input. Every field but Advice is
// set: a signal without its source and observation date is not published.
type Signal struct {
	// Code is the stable snake_case identifier of the rule, such as
	// "amf_blacklist". It is part of the public contract.
	Code string
	// Level is the severity of the signal.
	Level Level
	// Message is one factual French sentence stating what was observed.
	Message string
	// Advice is an optional French sentence telling the user what to do.
	Advice string
	// Source is the upstream source the fact comes from.
	Source Source
	// SourceURL is the public page where the fact can be checked.
	SourceURL string
	// ObservedAt is when the fact was read from the source.
	ObservedAt time.Time
}
