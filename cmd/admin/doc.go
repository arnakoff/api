// SPDX-License-Identifier: AGPL-3.0-only

// Command admin administers the API key store (ADR 0004).
//
//	admin migrate
//	admin key create -label <holder> [-env live|test]
//	admin key list
//	admin key revoke -id <id>
//
// migrate applies the embedded SQL migrations not yet applied, under an
// advisory lock: run it with a role that may change the schema, distinct
// from the role of the API, before deploying a version that needs them.
// The API never migrates on start.
//
// key create draws a key, "ark_live_…" or "ark_test_…", stores its keyed
// digest and prints the key once on standard output: it is never stored and
// cannot be shown again. key list prints every key, its hint (the last four
// characters), its dates and its label, never the key. key revoke revokes a
// key in the store at once; revoking it again keeps the first date. A
// running API keeps a valid key in memory for 30 seconds (app.KeyCacheTTL)
// and so may accept a revoked key for up to that long.
//
// The database is ARNAKOFF_DATABASE_URL, which must use
// sslmode=verify-full beyond the local machine; keys are digested with
// ARNAKOFF_API_KEY_SECRET, which must differ from ARNAKOFF_LOG_KEY, and
// issued for ARNAKOFF_API_KEY_ENV ("live" by default) unless -env says
// otherwise. Logs go to standard error.
package main
