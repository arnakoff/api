// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import (
	"errors"
	"fmt"

	"github.com/arnakoff/api/internal/app"
)

var (
	// ErrNotFound means the API returned no company with this SIREN: it is
	// unknown, or not open to the public (non-diffusible).
	ErrNotFound = fmt.Errorf("company not found in SIRENE: %w", app.ErrNotFound)

	// ErrThrottled means no request was sent: the API asked to wait
	// (Retry-After) and the wait is not over, or the request rate would
	// have to wait past the context's deadline.
	ErrThrottled = errors.New("SIRENE request rate exceeded")

	// ErrResponse means the API's answer is not a search result that can be
	// read.
	ErrResponse = errors.New("invalid SIRENE search response")
)
