// SPDX-License-Identifier: AGPL-3.0-only

// Package startup is what every command does before its own work: load the
// typed configuration, build the redacting logger from it and make that
// logger the log/slog default, so no line written by the command or a
// library bypasses the masking of personal data (ADR 0002).
//
// A configuration that cannot be used stops the command before anything
// else runs, with a non-zero exit code and an error that never quotes a
// secret.
//
// Every command logs to standard error: the MCP stdio transport allows a
// server to log there and reserves standard output for protocol messages
// (MCP specification 2025-06-18, Transports, stdio). The other commands
// follow for consistency.
package startup
