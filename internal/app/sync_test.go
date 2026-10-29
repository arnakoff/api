// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

var (
	errUpstream = errors.New("upstream down")
	errDisk     = errors.New("disk full")
)

func newSync(t *testing.T, st *store, fetchers ...app.SourceFetcher) (*app.Sync, *clock) {
	t.Helper()
	clk := &clock{now: t0}
	s, err := app.NewSync(st, clk, fetchers...)
	if err != nil {
		t.Fatalf("NewSync: %v", err)
	}
	return s, clk
}

// state returns the sync state of source.
func state(t *testing.T, s *app.Sync, source domain.Source) app.SourceSync {
	t.Helper()
	for _, st := range s.States() {
		if st.Source == source {
			return st
		}
	}
	t.Fatalf("no state for %s", source)
	return app.SourceSync{}
}

func TestSyncServesAValidSnapshotAndStoresIt(t *testing.T) {
	t.Parallel()

	newest := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	st := newStore()
	s, _ := newSync(t, st, newFetcher(amfSpec).then(snapshot(100, newest), nil))

	if err := s.RunSource(t.Context(), "AMF"); err != nil {
		t.Fatalf("RunSource: %v", err)
	}
	got, ok := s.Snapshot("AMF")
	if !ok || len(got.Rows) != 100 || !got.FetchedAt.Equal(t0) {
		t.Errorf("Snapshot() = %d rows fetched at %v, %t; want 100 rows fetched at %v", len(got.Rows), got.FetchedAt, ok, t0)
	}
	if saved := st.saved["AMF"]; len(saved.Rows) != 100 || !saved.FetchedAt.Equal(t0) {
		t.Errorf("stored %d rows fetched at %v, want 100 at %v", len(saved.Rows), saved.FetchedAt, t0)
	}
	want := app.SourceSync{Source: "AMF", LastSuccess: t0, Rows: 100, Newest: newest}
	if got := state(t, s, "AMF"); got != want {
		t.Errorf("state = %+v, want %+v", got, want)
	}
}

func TestSyncRejectsAnInvalidSnapshotAndKeepsTheLastValidOne(t *testing.T) {
	t.Parallel()

	noColumn := snapshot(100, time.Time{})
	noColumn.Header = []string{"nom"}
	tests := []struct {
		name string
		next domain.Snapshot
		err  error
		want error
	}{
		{"empty file", snapshot(0, time.Time{}), nil, domain.ErrSnapshotEmpty},
		{"missing column", noColumn, nil, domain.ErrSnapshotColumnMissing},
		{"30 % drop", snapshot(70, time.Time{}), nil, domain.ErrSnapshotShrunk},
		{"fetch failure", domain.Snapshot{}, errUpstream, errUpstream},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st := newStore()
			f := newFetcher(amfSpec).then(snapshot(100, time.Time{}), nil).then(tt.next, tt.err)
			s, clk := newSync(t, st, f)
			if err := s.RunSource(t.Context(), "AMF"); err != nil {
				t.Fatalf("first RunSource: %v", err)
			}
			clk.advance(time.Hour)

			err := s.RunSource(t.Context(), "AMF")
			if !errors.Is(err, tt.want) {
				t.Fatalf("RunSource() = %v, want %v", err, tt.want)
			}
			assertFirstSnapshotKept(t, s, st, tt.want)
		})
	}
}

// assertFirstSnapshotKept checks that the 100-row snapshot fetched at t0 is
// still served and stored after a sync an hour later failed with want.
func assertFirstSnapshotKept(t *testing.T, s *app.Sync, st *store, want error) {
	t.Helper()
	if got, _ := s.Snapshot("AMF"); len(got.Rows) != 100 || !got.FetchedAt.Equal(t0) {
		t.Errorf("Snapshot() = %d rows from %v, want the 100 rows from %v", len(got.Rows), got.FetchedAt, t0)
	}
	if saved := st.saved["AMF"]; !saved.FetchedAt.Equal(t0) {
		t.Errorf("stored the snapshot fetched at %v, want only the one from %v", saved.FetchedAt, t0)
	}
	got := state(t, s, "AMF")
	if !got.LastSuccess.Equal(t0) || !got.LastFailure.Equal(t0.Add(time.Hour)) || !errors.Is(got.LastError, want) || got.Rows != 100 {
		t.Errorf("state = %+v, want the success at %v and the failure an hour later", got, t0)
	}
}

func TestSyncKeepsServingWhenTheStoreFails(t *testing.T) {
	t.Parallel()

	st := newStore()
	st.saveErr = errDisk
	s, _ := newSync(t, st, newFetcher(amfSpec).then(snapshot(100, time.Time{}), nil))

	if err := s.RunSource(t.Context(), "AMF"); !errors.Is(err, errDisk) {
		t.Fatalf("RunSource() = %v, want %v", err, errDisk)
	}
	if _, ok := s.Snapshot("AMF"); ok {
		t.Error("Snapshot() served a snapshot the store did not keep")
	}
}

func TestSyncKeepsTheLastErrorAfterASuccess(t *testing.T) {
	t.Parallel()

	f := newFetcher(amfSpec).then(domain.Snapshot{}, errUpstream).then(snapshot(10, time.Time{}), nil)
	s, clk := newSync(t, newStore(), f)
	_ = s.RunSource(t.Context(), "AMF")
	clk.advance(time.Hour)
	if err := s.RunSource(t.Context(), "AMF"); err != nil {
		t.Fatalf("RunSource: %v", err)
	}

	got := state(t, s, "AMF")
	if !got.LastFailure.Equal(t0) || !errors.Is(got.LastError, errUpstream) || !got.LastSuccess.Equal(t0.Add(time.Hour)) {
		t.Errorf("state = %+v, want the failure at %v kept beside the later success", got, t0)
	}
}

func TestSyncRunSyncsEverySourceAndJoinsTheErrors(t *testing.T) {
	t.Parallel()

	acpr := domain.SnapshotSpec{Source: "ACPR", Columns: []string{"nom"}}
	arcep := domain.SnapshotSpec{Source: "ARCEP", Columns: []string{"nom"}}
	acprSnap := snapshot(10, time.Time{})
	acprSnap.Source = "ACPR"
	s, _ := newSync(t, newStore(),
		newFetcher(amfSpec).then(domain.Snapshot{}, errUpstream),
		newFetcher(acpr).then(acprSnap, nil),
		newFetcher(arcep).then(domain.Snapshot{}, errDisk),
	)

	err := s.Run(t.Context())
	if !errors.Is(err, errUpstream) || !errors.Is(err, errDisk) {
		t.Errorf("Run() = %v, want both failures", err)
	}
	if _, ok := s.Snapshot("ACPR"); !ok {
		t.Error("Run() did not sync ACPR after the AMF failure")
	}
	var order []domain.Source
	for _, st := range s.States() {
		order = append(order, st.Source)
	}
	if len(order) != 3 || order[0] != "AMF" || order[1] != "ACPR" || order[2] != "ARCEP" {
		t.Errorf("States() order = %v, want the declaration order", order)
	}
}

func TestSyncRejectsUnknownAndDuplicateSources(t *testing.T) {
	t.Parallel()

	s, _ := newSync(t, newStore(), newFetcher(amfSpec))
	if err := s.RunSource(t.Context(), "BODACC"); !errors.Is(err, app.ErrUnknownSource) {
		t.Errorf("RunSource(BODACC) = %v, want %v", err, app.ErrUnknownSource)
	}
	if _, ok := s.Snapshot("BODACC"); ok {
		t.Error("Snapshot(BODACC) found a snapshot")
	}
	_, err := app.NewSync(newStore(), &clock{}, newFetcher(amfSpec), newFetcher(amfSpec))
	if !errors.Is(err, app.ErrDuplicateSource) {
		t.Errorf("NewSync(AMF, AMF) = %v, want %v", err, app.ErrDuplicateSource)
	}
}

func TestSyncHasNoSnapshotBeforeTheFirstRun(t *testing.T) {
	t.Parallel()

	s, _ := newSync(t, newStore(), newFetcher(amfSpec))
	if _, ok := s.Snapshot("AMF"); ok {
		t.Error("Snapshot() found a snapshot before any run")
	}
	if got := state(t, s, "AMF"); got != (app.SourceSync{Source: "AMF"}) {
		t.Errorf("state = %+v, want only the source", got)
	}
}

func TestSyncReadersSeeWholeSnapshotsDuringSyncs(t *testing.T) {
	t.Parallel()

	const syncs = 50
	f := newFetcher(amfSpec)
	for i := range syncs {
		f.then(snapshot(100+i, time.Time{}), nil)
	}
	s, clk := newSync(t, newStore(), f)

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range syncs {
				readServed(t, s)
			}
		})
	}
	for range syncs {
		clk.advance(time.Minute)
		if err := s.RunSource(t.Context(), "AMF"); err != nil {
			t.Errorf("RunSource: %v", err)
		}
	}
	wg.Wait()
}

// readServed reads the AMF snapshot and state of s and checks that a served
// snapshot is a whole one, of 100 rows at least.
func readServed(t *testing.T, s *app.Sync) {
	t.Helper()
	snap, ok := s.Snapshot("AMF")
	if !ok {
		return
	}
	if len(snap.Rows) < 100 {
		t.Errorf("Snapshot() = %d rows, want a whole snapshot", len(snap.Rows))
	}
	if st := s.States()[0]; st.Rows < 100 {
		t.Errorf("state has %d rows while a snapshot is served", st.Rows)
	}
}

func TestSyncStopsOnACancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	s, _ := newSync(t, newStore(), newFetcher(amfSpec).then(snapshot(10, time.Time{}), nil))
	if err := s.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Run() = %v, want %v", err, context.Canceled)
	}
}
