// SPDX-License-Identifier: AGPL-3.0-only

// Package replay records the answers of the registries and plays them back,
// so that the evaluation corpus is scored without the network.
//
// A Tape holds, for each registry, the answers it gave keyed by what it was
// asked: the registrable domain for RDAP, the SIREN for SIRENE and BODACC.
// Only answers are recorded (a value, "not found" or "not covered"), never a
// failure, and only the values the app reads: the decoded dates, names and
// notice references, never a raw upstream response, so no contact is kept.
//
// # Files
//
// A tape is a directory of three UTF-8 JSON files, rdap.json, sirene.json
// and bodacc.json, each an object from key to entry. An entry holds either
// "value" or "answer", which is "not_found" or "not_covered". A missing file
// is an empty track.
package replay
