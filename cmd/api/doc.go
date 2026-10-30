// SPDX-License-Identifier: AGPL-3.0-only

// Command api serves the public Arnakoff HTTP API.
//
// It serves the snapshots the sync command stores: it restores them at
// start, then looks for new ones every minute and swaps them in without a
// restart. It downloads nothing itself. A stored snapshot that cannot be
// reloaded is logged once, until it reloads again.
package main
