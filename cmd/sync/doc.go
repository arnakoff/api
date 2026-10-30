// SPDX-License-Identifier: AGPL-3.0-only

// Command sync downloads the upstream sources and stores validated snapshots.
//
// It first restores the stored snapshots, so that a new file is checked
// against the last valid one (no more than 20 % fewer rows) and a source
// synced recently is not downloaded again. Then, by default, it syncs each
// source at its interval plus up to a tenth of it at random, until SIGINT or
// SIGTERM. A failed or rejected sync keeps the last valid snapshot in
// service and raises an alert: an error line in the log and, when
// ARNAKOFF_ALERT_WEBHOOK is set, a JSON POST to that URL.
//
// Usage:
//
//	sync [-once [-dry-run] [-only <source>]]
//
// Flags:
//
//	-once      sync every source once, then exit
//	-dry-run   with -once: download and validate, store nothing, send no webhook
//	-only      with -once: sync this source only, by its exact name ("AMF PSAN")
//
// Exit codes:
//
//	0  every source synced, or SIGINT or SIGTERM received
//	1  invalid configuration, unknown source, or a source failed or was rejected
//	2  invalid flags
package main
