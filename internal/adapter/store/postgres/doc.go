// SPDX-License-Identifier: AGPL-3.0-only

// Package postgres keeps the records of the API keys in PostgreSQL (ADR
// 0004), through jackc/pgx: their keyed digests, never the keys.
//
// The schema is changed by versioned SQL migrations embedded in the binary
// and applied by Migrate, from the admin command with a role that may
// change it; the API, connected with a role that can only read and write
// the tables, never migrates. A connection beyond the local machine must
// verify the server's certificate and host name (sslmode=verify-full), and
// every query is bounded by queryTimeout on top of its context.
package postgres
