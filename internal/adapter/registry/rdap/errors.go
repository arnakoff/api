// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import "errors"

var (
	// ErrBootstrap means the IANA bootstrap file could not be read.
	ErrBootstrap = errors.New("unusable RDAP bootstrap file")

	// ErrNoServer means the bootstrap file names no HTTPS RDAP server for
	// the domain's extension.
	ErrNoServer = errors.New("no RDAP server for this extension")

	// ErrNotFound means the registry answered that it holds no such domain.
	ErrNotFound = errors.New("domain not found in RDAP")

	// ErrThrottled means the registry asked to wait (Retry-After) and the
	// wait is not over. No request was sent.
	ErrThrottled = errors.New("RDAP server asked to wait")

	// ErrResponse means the registry's answer is not an RDAP domain object.
	ErrResponse = errors.New("invalid RDAP domain response")
)
