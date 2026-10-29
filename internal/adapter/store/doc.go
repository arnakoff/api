// SPDX-License-Identifier: AGPL-3.0-only

// Package store keeps the last valid snapshot of each source on disk, shared
// by the sync and API processes and kept across restarts.
//
// # File format, version 1
//
// Each source has one file, <name>.snapshot, where name is the source
// lowercased with every run of characters other than a-z and 0-9 turned into
// a single "-" ("AMF listes blanches" is amf-listes-blanches.snapshot). The
// file holds two parts, both UTF-8 JSON:
//
//   - an envelope on the first line, ended by "\n":
//     {"format":"arnakoff-snapshot","version":1,"sha256":"<hex>"}, where
//     sha256 is the hex SHA-256 digest of the body;
//   - the body, the rest of the file: {"source":"AMF",
//     "fetched_at":"<RFC 3339>","newest":"<RFC 3339>","header":[...],
//     "rows":[[...],...]}; newest is left out when the source dates no entry.
//
// A file whose envelope, digest or body does not check out, or that holds
// another source, is rejected with ErrCorrupt; one of another version with
// ErrVersion. A change to the body or envelope bumps the version.
//
// A file is written to a temporary file in the same directory, synced, then
// renamed over the previous one, and the directory is synced on Unix: a
// reader sees the previous file or the new one, never part of one.
package store
