// SPDX-License-Identifier: AGPL-3.0-only

// Command api serves the public Arnakoff HTTP API.
//
// It listens on ARNAKOFF_ADDR (":8100" by default) and stops cleanly on
// SIGINT or SIGTERM, letting in-flight requests finish. /healthz answers
// while the process runs, /readyz once every snapshot is loaded.
//
// GET /v1/check reads the lists of the snapshots the sync command stores
// and queries RDAP, SIRENE and BODACC by identifier, for at most ten
// seconds a check. The snapshots are restored at start, then looked for
// every minute and swapped in without a restart; the lists themselves are
// never downloaded here. A stored snapshot that cannot be reloaded is
// logged once, until it reloads again.
//
// Its errors are RFC 9457 problems typed under ARNAKOFF_PROBLEM_BASE_URL,
// the documentation of the errors on the site.
package main
