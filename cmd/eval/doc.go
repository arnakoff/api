// SPDX-License-Identifier: AGPL-3.0-only

// Command eval measures signal quality against the labelled corpus of
// testdata/corpus (scam.csv and legit.csv) and writes the eval-signals
// table to standard output.
//
// It reads the AMF and ACPR blacklists and the AMF white lists from the
// snapshots cmd/sync stored in ARNAKOFF_DATA_DIR/snapshots, as the API
// serves them; a list without a stored snapshot is reported unavailable.
//
// Online, the default, it asks the registries (RDAP, SIRENE, BODACC) and
// records their answers to <corpus>/replay: an answer replaces the one
// recorded for the same query, a failure keeps it. With
// -offline it replays that recording and reaches no network; a query the
// recording lacks leaves its source unavailable, which the report counts.
//
//	go run ./cmd/eval -offline
//	go run ./cmd/eval -offline -without amf,acpr
//
// Flags:
//
//	-corpus dir     corpus directory (default testdata/corpus)
//	-offline        replay the recorded registry answers
//	-without list   comma-separated sources to leave out: amf, acpr,
//	                amf psan, amf biens divers, rdap, sirene, bodacc
package main
