// SPDX-License-Identifier: AGPL-3.0-only

// Package config reads the configuration of every command from environment
// variables into one typed Config, validated as a whole at startup so a
// command stops before doing anything with a value it cannot use
// (ADR 0002).
//
// The variables, their defaults and an example are in .env.example. A
// secret, such as the log key, is held as a Secret: it never appears in a
// log line, an error message or a formatted Config.
//
// Commands write only under Config.DataDir, the data volume, so the rest of
// the file system can be mounted read-only.
package config
