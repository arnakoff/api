// SPDX-License-Identifier: AGPL-3.0-only

// Command loadtest serves the API as cmd/api does, for a load test on this
// machine: same handler, same lists read from the stored snapshots, same
// API keys checked in PostgreSQL, same rate limits. Only the registries
// differ: RDAP, SIRENE and BODACC are replayed from the corpus tape
// (testdata/corpus/replay), each answer after -latency, so no load ever
// reaches them. docs/performance.md tells how to run it with vegeta and
// records the figures.
//
// It reads the configuration of cmd/api (ARNAKOFF_ADDR,
// ARNAKOFF_DATABASE_URL, ARNAKOFF_API_KEY_SECRET, ARNAKOFF_RATE_LIMIT_KEY…)
// and also serves the runtime profiles under /debug/pprof/ on -pprof, a
// loopback address only. It never runs in production.
//
// With -targets it writes instead, in the vegeta http format, one
// GET /v1/check per corpus input against a loopback base URL, and stops.
//
//	go run ./cmd/loadtest -targets http://127.0.0.1:8100 > var/load/targets.http
//	go run ./cmd/loadtest -latency 200ms
//
// Flags:
//
//	-corpus dir      corpus directory (default testdata/corpus)
//	-latency d       time each replayed registry answer takes (default 0)
//	-pprof addr      loopback host:port of the profiles, empty for none
//	                 (default 127.0.0.1:6060)
//	-targets url     write the targets for this loopback base URL, then stop
package main
