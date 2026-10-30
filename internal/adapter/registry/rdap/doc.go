// SPDX-License-Identifier: AGPL-3.0-only

// Package rdap reads the registration date of a domain from the RDAP
// server of its registry (RFC 9082, RFC 9083), found through the IANA
// bootstrap file (RFC 9224), never through a third-party aggregator
// (docs/sources.md).
//
// Only the registrable domain is sent, and only the domain object's own
// registration and last changed events are read: entities, and so the
// contacts they hold, are never decoded. Answers are cached for a day,
// "not found" and "no server" for a few minutes, and a registry that sends
// a Retry-After is left alone for that long.
package rdap
