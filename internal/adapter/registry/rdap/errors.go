// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"errors"
	"fmt"

	"github.com/arnakoff/api/internal/app"
)

var (
	// ErrBootstrap means the IANA bootstrap file could not be read.
	ErrBootstrap = errors.New("unusable RDAP bootstrap file")

	// ErrNoServer means the bootstrap file names no HTTPS RDAP server for
	// the domain's extension.
	ErrNoServer = fmt.Errorf("no RDAP server for this extension: %w", app.ErrNotCovered)

	// ErrNotFound means the registry answered that it holds no such domain.
	ErrNotFound = fmt.Errorf("domain not found in RDAP: %w", app.ErrNotFound)

	// ErrThrottled means the registry asked to wait (Retry-After) and the
	// wait is not over. No request was sent.
	ErrThrottled = errors.New("RDAP server asked to wait")

	// ErrResponse means the registry's answer is not an RDAP domain object.
	ErrResponse = errors.New("invalid RDAP domain response")
)
