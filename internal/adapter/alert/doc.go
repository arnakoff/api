// SPDX-License-Identifier: AGPL-3.0-only

// Package alert tells the operator that a source could not be synced or
// reloaded: an error line in the log, and a JSON POST to a webhook when one
// is configured.
//
// # Webhook body
//
// One JSON object: {"text":"arnakoff: AMF update failed: <error>",
// "source":"AMF","at":"<RFC 3339>","error":"<error>"}. text reads on its
// own in a chat channel; the other fields are for a machine. The error
// names the source and why, never a personal datum: connectors quote no
// value of the files they parse.
package alert
