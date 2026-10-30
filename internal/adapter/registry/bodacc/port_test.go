// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import (
	"testing"

	"github.com/arnakoff/api/internal/app"
)

// The client is the app's notice registry.
var _ app.NoticeRegistry = (*Client)(nil)

func TestClientSource(t *testing.T) {
	t.Parallel()

	if got := (&Client{}).Source(); got != "BODACC" {
		t.Errorf("Source() = %q, want BODACC", got)
	}
}
