// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

// Registration holds the dates a domain registry publishes about a
// registrable domain, and nothing about its holder. A zero time means the
// registry did not publish that date.
type Registration struct {
	// Registered is when the domain was first registered.
	Registered time.Time
	// LastChanged is when the registry last changed the domain's record.
	LastChanged time.Time
}
