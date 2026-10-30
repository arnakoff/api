// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"slices"
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

// acprSpec is the spec of a second fake source.
var acprSpec = domain.SnapshotSpec{Source: "ACPR", Columns: []string{"nom"}}

// acprSnapshot returns an ACPR snapshot of n rows.
func acprSnapshot(n int) domain.Snapshot {
	s := snapshot(n, time.Time{})
	s.Source = "ACPR"
	return s
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

func (s *store) setLoadErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadErr = err
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

// errStop ends a scheduling loop: the sleeper returns it when its stop
// predicate says so.
var errStop = errors.New("sleeper stopped")

// sleeper records each pause asked and grants it, moving the clock past it,
// until stop reports true for a pause, by index and duration; it returns
// errStop for that one. before, when set, runs before each granted pause
// with its index.
type sleeper struct {
	clock  *clock
	stop   func(pause int, d time.Duration) bool
	before func(pause int)

	mu      sync.Mutex
	asked   []time.Duration
	granted []time.Duration
}

// after stops at pause n: it grants n pauses.
func after(n int) func(int, time.Duration) bool {
	return func(pause int, _ time.Duration) bool { return pause >= n }
}

// immediateOnly grants every zero pause and stops at the first other one.
func immediateOnly(_ int, d time.Duration) bool { return d != 0 }

func (s *sleeper) Sleep(ctx context.Context, d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	pause := len(s.asked)
	s.asked = append(s.asked, d)
	if s.stop(pause, d) {
		return errStop
	}
	if s.before != nil {
		s.before(pause)
	}
	s.granted = append(s.granted, d)
	s.clock.advance(d)
	return nil
}

// pauses returns the pauses granted.
func (s *sleeper) pauses() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.granted)
}

// firstAsked returns the first pause asked, granted or not.
func (s *sleeper) firstAsked() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.asked[0]
}

// alerter records the alerts it receives.
type alerter struct {
	mu     sync.Mutex
	alerts []app.Alert
}

func (a *alerter) Alert(_ context.Context, al app.Alert) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.alerts = append(a.alerts, al)
}

func (a *alerter) received() []app.Alert {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.alerts)
}

// fixedJitter returns a jitter that always adds d, and records the largest
// jitter it was asked for.
func fixedJitter(d time.Duration, asked *time.Duration) app.Jitter {
	return func(maxJitter time.Duration) time.Duration {
		*asked = maxJitter
		return d
	}
}
