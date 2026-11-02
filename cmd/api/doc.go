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
// POST /v1/check/batch checks up to ARNAKOFF_BATCH_MAX_ITEMS inputs (50 by
// default) in one request, a few at a time, within the time of one check:
// an input still waiting then has its registries reported unavailable.
//
// Both check routes require an API key, "Authorization: Bearer ark_live_…"
// (RFC 6750): its keyed digest is looked up in the PostgreSQL database of
// ARNAKOFF_DATABASE_URL under ARNAKOFF_API_KEY_SECRET, and only keys of
// ARNAKOFF_API_KEY_ENV ("live" by default) are accepted. Both variables are
// required to start. The database is not reached at start: while it is
// unreachable, the check routes answer 503, never unchecked. Keys are issued
// and the schema migrated by the admin command.
//
// Each API key may send ARNAKOFF_RATE_LIMIT_KEY requests a minute to the
// check routes (60 by default), and each client address
// ARNAKOFF_RATE_LIMIT_IP to the routes needing no key (30 by default); the
// probes are never limited. Each client address may also send
// ARNAKOFF_RATE_LIMIT_AUTH refused keys a minute to the check routes (30 by
// default): over it, its keys are answered 429 without being looked up, so
// a flood of unknown keys cannot hold the key database. The limits are kept
// in memory, per instance.
// Behind a proxy, ARNAKOFF_TRUSTED_PROXIES lists its CIDR prefixes, whose
// X-Forwarded-For field then tells the client's address; "none" says no
// proxy stands in front. It is required to start with live keys.
//
// Its errors are RFC 9457 problems typed under ARNAKOFF_PROBLEM_BASE_URL,
// the documentation of the errors on the site.
package main
