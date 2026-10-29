// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// amfSpec is the spec of the fake source most tests sync.
var amfSpec = domain.SnapshotSpec{
	Source:  "AMF",
	Columns: []string{"nom", "date_inscription"},
	MinRows: 1,
}

// t0 is the fixed time the tests start at.
var t0 = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

// snapshot returns an AMF snapshot of n rows whose newest entry is newest.
func snapshot(n int, newest time.Time) domain.Snapshot {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{"site-" + strconv.Itoa(i) + ".example", "2026-10-29"}
	}
	return domain.Snapshot{
		Source: "AMF",
		Header: []string{"nom", "categorie", "date_inscription"},
		Rows:   rows,
		Newest: newest,
	}
}

// clock is a settable time source.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// fetcher answers each Fetch with the next queued answer.
type fetcher struct {
	spec domain.SnapshotSpec

	mu      sync.Mutex
	answers []fetchAnswer
}

type fetchAnswer struct {
	snapshot domain.Snapshot
	err      error
}

func newFetcher(spec domain.SnapshotSpec) *fetcher { return &fetcher{spec: spec} }

func (f *fetcher) then(s domain.Snapshot, err error) *fetcher {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answers = append(f.answers, fetchAnswer{s, err})
	return f
}

func (f *fetcher) Spec() domain.SnapshotSpec { return f.spec }

func (f *fetcher) Fetch(context.Context) (domain.Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a := f.answers[0]
	f.answers = f.answers[1:]
	return a.snapshot, a.err
}

// store keeps snapshots in memory and fails every Save with saveErr.
type store struct {
	mu      sync.Mutex
	saved   map[domain.Source]domain.Snapshot
	saveErr error
	loadErr error
}

func newStore() *store { return &store{saved: make(map[domain.Source]domain.Snapshot)} }

func (s *store) Save(_ context.Context, snap domain.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved[snap.Source] = snap
	return nil
}

func (s *store) Load(_ context.Context, source domain.Source) (domain.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return domain.Snapshot{}, s.loadErr
	}
	snap, ok := s.saved[source]
	if !ok {
		return domain.Snapshot{}, app.ErrNoSnapshot
	}
	return snap, nil
}
