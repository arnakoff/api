// SPDX-License-Identifier: AGPL-3.0-only

// Command signaldoc writes the table of docs/signals.md from the signal
// catalog (internal/domain/signal), between the markers of
// internal/adapter/signaldoc; the rest of the file is left as it is.
// go generate runs it:
//
//	go generate ./...
//
// Flags:
//
//	-file path   the document to rewrite (default docs/signals.md)
package main
