// SPDX-License-Identifier: AGPL-3.0-only

// Package httpapi serves the public HTTP API, following its contract
// (api/openapi.yaml): the server and its timeouts, the probes, and the
// middleware every request goes through (request identifier, access log,
// panic recovery, security headers). It never serves HTML: every error is
// an RFC 9457 problem.
package httpapi
