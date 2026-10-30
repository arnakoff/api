// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/app"
)

// The client is the app's domain registry.
var _ app.DomainRegistry = (*Client)(nil)

func TestErrorsMatchTheAppRegistryErrors(t *testing.T) {
	t.Parallel()

	if !errors.Is(ErrNotFound, app.ErrNotFound) {
		t.Error("ErrNotFound does not match app.ErrNotFound")
	}
	if !errors.Is(ErrNoServer, app.ErrNotCovered) {
		t.Error("ErrNoServer does not match app.ErrNotCovered")
	}
	for _, err := range []error{ErrBootstrap, ErrThrottled, ErrResponse} {
		if errors.Is(err, app.ErrNotFound) || errors.Is(err, app.ErrNotCovered) {
			t.Errorf("%v matches an answer, want a failure", err)
		}
	}
}

func TestClientSource(t *testing.T) {
	t.Parallel()

	if got := (&Client{}).Source(); got != "RDAP" {
		t.Errorf("Source() = %q, want RDAP", got)
	}
}
