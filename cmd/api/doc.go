// SPDX-License-Identifier: AGPL-3.0-only

// Command api serves the public Arnakoff HTTP API.
//
// It listens on ARNAKOFF_ADDR (":8100" by default) and stops cleanly on
// SIGINT or SIGTERM, letting in-flight requests finish. /healthz answers
// while the process runs, /readyz once every snapshot is loaded.
//
// It serves the snapshots the sync command stores: it restores them at
// start, then looks for new ones every minute and swaps them in without a
// restart. It downloads nothing itself. A stored snapshot that cannot be
// reloaded is logged once, until it reloads again.
package main
