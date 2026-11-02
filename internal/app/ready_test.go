// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func TestSyncIsReadyOnceEverySourceHasASnapshot(t *testing.T) {
	t.Parallel()

	newest := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	s, _ := newSync(t, newStore(),
		newFetcher(amfSpec).then(snapshot(100, newest), nil),
		newFetcher(acprSpec).then(domain.Snapshot{}, errUpstream).then(acprSnapshot(10), nil))

	if s.Ready() {
		t.Error("Ready() = true before any sync")
	}
	_ = s.Run(t.Context())
	if s.Ready() {
		t.Error("Ready() = true while ACPR has no snapshot")
	}
	_ = s.RunSource(t.Context(), "ACPR")
	if !s.Ready() {
		t.Error("Ready() = false once every source has a snapshot")
	}
}
