// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import (
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/app"
)

// The client is the app's company registry.
var _ app.CompanyRegistry = (*Client)(nil)

func TestErrorsMatchTheAppRegistryErrors(t *testing.T) {
	t.Parallel()

	if !errors.Is(ErrNotFound, app.ErrNotFound) {
		t.Error("ErrNotFound does not match app.ErrNotFound")
	}
	for _, err := range []error{ErrThrottled, ErrResponse} {
		if errors.Is(err, app.ErrNotFound) || errors.Is(err, app.ErrNotCovered) {
			t.Errorf("%v matches an answer, want a failure", err)
		}
	}
}

func TestClientSource(t *testing.T) {
	t.Parallel()

	if got := (&Client{}).Source(); got != "SIRENE" {
		t.Errorf("Source() = %q, want SIRENE", got)
	}
}
