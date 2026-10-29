// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// stored returns an AMF snapshot of n rows fetched at fetchedAt.
func stored(n int, fetchedAt time.Time) domain.Snapshot {
	s := snapshot(n, fetchedAt.Add(-24*time.Hour))
	s.FetchedAt = fetchedAt
	return s
}

func TestRestoreServesTheStoredSnapshot(t *testing.T) {
	t.Parallel()

	st := newStore()
	st.saved["AMF"] = stored(50, t0.Add(-time.Hour))
	s, _ := newSync(t, st, newFetcher(amfSpec))

	if err := s.Restore(t.Context()); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got, ok := s.Snapshot("AMF"); !ok || len(got.Rows) != 50 {
		t.Errorf("Snapshot() = %d rows, %t; want the 50 stored rows", len(got.Rows), ok)
	}
	got := state(t, s, "AMF")
	if !got.LastSuccess.Equal(t0.Add(-time.Hour)) || got.Rows != 50 || !got.Newest.Equal(t0.Add(-25*time.Hour)) {
		t.Errorf("state = %+v, want the stored snapshot's figures", got)
	}
}

func TestRestoreWithoutStoredSnapshotServesNothing(t *testing.T) {
	t.Parallel()

	s, _ := newSync(t, newStore(), newFetcher(amfSpec))
	if err := s.Restore(t.Context()); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if _, ok := s.Snapshot("AMF"); ok {
		t.Error("Snapshot() found a snapshot the store does not hold")
	}
}

func TestRestoreSwapsOnlyANewerSnapshot(t *testing.T) {
	t.Parallel()

	st := newStore()
	s, clk := newSync(t, st, newFetcher(amfSpec).then(snapshot(100, time.Time{}), nil))
	if err := s.RunSource(t.Context(), "AMF"); err != nil {
		t.Fatalf("RunSource: %v", err)
	}

	st.saved["AMF"] = stored(40, t0.Add(-time.Hour))
	if err := s.Restore(t.Context()); err != nil {
		t.Fatalf("Restore older: %v", err)
	}
	if got, _ := s.Snapshot("AMF"); len(got.Rows) != 100 {
		t.Errorf("Restore() replaced the served snapshot with an older one: %d rows", len(got.Rows))
	}

	clk.advance(2 * time.Hour)
	st.saved["AMF"] = stored(90, t0.Add(time.Hour))
	if err := s.Restore(t.Context()); err != nil {
		t.Fatalf("Restore newer: %v", err)
	}
	if got, _ := s.Snapshot("AMF"); len(got.Rows) != 90 {
		t.Errorf("Snapshot() = %d rows, want the 90 rows of the newer stored snapshot", len(got.Rows))
	}
}

func TestRestoreRejectsAnInvalidStoredSnapshot(t *testing.T) {
	t.Parallel()

	st := newStore()
	bad := stored(10, t0)
	bad.Header = []string{"nom"}
	st.saved["AMF"] = bad
	s, _ := newSync(t, st, newFetcher(amfSpec))

	if err := s.Restore(t.Context()); !errors.Is(err, domain.ErrSnapshotColumnMissing) {
		t.Fatalf("Restore() = %v, want %v", err, domain.ErrSnapshotColumnMissing)
	}
	if _, ok := s.Snapshot("AMF"); ok {
		t.Error("Snapshot() serves an invalid stored snapshot")
	}
}

func TestRestoreReportsStoreFailures(t *testing.T) {
	t.Parallel()

	st := newStore()
	st.loadErr = errDisk
	s, _ := newSync(t, st, newFetcher(amfSpec))
	if err := s.Restore(t.Context()); !errors.Is(err, errDisk) {
		t.Errorf("Restore() = %v, want %v", err, errDisk)
	}
}
