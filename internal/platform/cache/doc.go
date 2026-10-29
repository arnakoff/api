// SPDX-License-Identifier: AGPL-3.0-only

// Package cache keeps registry answers in memory for a while, so a burst of
// checks on the same identifier calls an upstream once (ADR 0002).
//
// A Cache holds at most Config.MaxEntries entries and evicts the least
// recently used one beyond that. A value lives for Config.TTL; an error
// lives for the shorter Config.NegativeTTL when Config.Negative says it is a
// lasting answer, such as "no such company", and is never kept otherwise.
// Concurrent misses on one key share a single load.
package cache
