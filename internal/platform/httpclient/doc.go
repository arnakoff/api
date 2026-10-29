// SPDX-License-Identifier: AGPL-3.0-only

// Package httpclient is the only way out to the network: every source and
// registry fetches through a Client (ADR 0002).
//
// A Client sends GET requests over HTTPS to an exact list of hosts, follows
// a redirect only to one of them, bounds every phase of a call with its own
// timeout, refuses a body over a size limit and retries 429 and 5xx answers
// with exponential backoff, full jitter and Retry-After.
//
// A Client never uses a proxy, even when HTTP_PROXY or HTTPS_PROXY is set:
// an inherited proxy would see every query and widen the egress surface
// without anyone choosing it. Filtering egress belongs to the network.
package httpclient
