// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
)

func TestDryRunValidatesAgainstTheStoreWithoutWritingIt(t *testing.T) {
	t.Parallel()

	st := newStore()
	st.saved["AMF"] = stored(100, t0.Add(-time.Hour))
	f := newFetcher(amfSpec).then(snapshot(50, time.Time{}), nil).then(snapshot(95, time.Time{}), nil)
	s, err := app.NewSync(app.DryRun(st), &clock{now: t0}, f)
	if err != nil {
		t.Fatalf("NewSync: %v", err)
	}
	if err := s.Restore(t.Context()); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if err := s.RunSource(t.Context(), "AMF"); err == nil {
		t.Error("RunSource() accepted a snapshot half the size of the stored one")
	}
	if err := s.RunSource(t.Context(), "AMF"); err != nil {
		t.Errorf("RunSource() = %v, want a valid snapshot accepted", err)
	}
	if got := st.saved["AMF"]; len(got.Rows) != 100 {
		t.Errorf("the store holds %d rows, want the 100 it held before", len(got.Rows))
	}
}
