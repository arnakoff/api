// SPDX-License-Identifier: AGPL-3.0-only

// Package api holds the public contract of the HTTP API, openapi.yaml
// (OpenAPI 3.1.2). Handlers follow it, and contract tests check their
// responses against it.
package api

import (
	"bytes"
	_ "embed"
)

//go:embed openapi.yaml
var spec []byte

// Spec returns the OpenAPI document of the API, in YAML. The slice is a
// copy: the caller may change it.
func Spec() []byte {
	return bytes.Clone(spec)
}
