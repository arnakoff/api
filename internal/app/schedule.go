// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// ErrInterval means a plan has no positive interval.
var ErrInterval = errors.New("sync interval must be positive")

// jitterShare is the largest jitter added to an interval, as a fraction of
// it: 1/10, so that sources and replicas started together drift apart.
const jitterShare = 10

// Plan says how often a source is synced.
type Plan struct {
	Source domain.Source
	Every  time.Duration
}

// Jitter returns a random duration from 0 up to, but not including, max.
type Jitter func(maxJitter time.Duration) time.Duration

// Scheduler syncs each planned source at its interval, plus a jitter, and
// alerts on every sync that fails or is rejected.
type Scheduler struct {
	sync    *Sync
	sleeper Sleeper
	alerter Alerter
	jitter  Jitter
	plans   []Plan
}

// NewScheduler returns a Scheduler of plans, or ErrUnknownSource,
// ErrDuplicateSource or ErrInterval for a plan s cannot follow.
func NewScheduler(s *Sync, sleeper Sleeper, alerter Alerter, jitter Jitter, plans ...Plan) (*Scheduler, error) {
	seen := make(map[domain.Source]bool, len(plans))
	for _, p := range plans {
		switch {
		case s.feed(p.Source) == nil:
			return nil, fmt.Errorf("%w: %s", ErrUnknownSource, p.Source)
		case seen[p.Source]:
			return nil, fmt.Errorf("%w: %s", ErrDuplicateSource, p.Source)
		case p.Every <= 0:
			return nil, fmt.Errorf("%w: %s every %v", ErrInterval, p.Source, p.Every)
		}
		seen[p.Source] = true
	}
	return &Scheduler{sync: s, sleeper: sleeper, alerter: alerter, jitter: jitter, plans: plans}, nil
}

// Run syncs every planned source until ctx is done. A
// source whose snapshot in service is younger than its interval waits until
// it is due: restore the stored snapshots first so that a restart does not
// download every source again.
func (sc *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, p := range sc.plans {
		wg.Go(func() { sc.loop(ctx, p) })
	}
	wg.Wait()
}

// Once syncs each of sources once, or every planned source when none is
// given, alerts on each failure and joins the errors. It returns
// ErrUnknownSource without syncing anything when a source is not planned.
func (sc *Scheduler) Once(ctx context.Context, sources ...domain.Source) error {
	if len(sources) == 0 {
		for _, p := range sc.plans {
			sources = append(sources, p.Source)
		}
	}
	for _, src := range sources {
		if !sc.planned(src) {
			return fmt.Errorf("%w: %s", ErrUnknownSource, src)
		}
	}
	var errs []error
	for _, src := range sources {
		errs = append(errs, sc.syncOnce(ctx, src))
	}
	return errors.Join(errs...)
}

// loop syncs p.Source each time it is due, until the sleeper stops.
func (sc *Scheduler) loop(ctx context.Context, p Plan) {
	for delay := sc.firstDelay(p); sc.sleeper.Sleep(ctx, delay) == nil; {
		_ = sc.syncOnce(ctx, p.Source) // Alerted; the loop goes on.
		delay = p.Every + sc.jitter(p.Every/jitterShare)
	}
}

// firstDelay is how long p.Source waits before its first sync: until its
// snapshot in service is due, at once when none is.
func (sc *Scheduler) firstDelay(p Plan) time.Duration {
	snap, ok := sc.sync.Snapshot(p.Source)
	if !ok {
		return 0
	}
	return max(snap.FetchedAt.Add(p.Every).Sub(sc.sync.clock.Now()), 0)
}

// syncOnce syncs source and alerts when it fails, unless ctx is done.
func (sc *Scheduler) syncOnce(ctx context.Context, source domain.Source) error {
	err := sc.sync.RunSource(ctx, source)
	if err != nil && ctx.Err() == nil {
		sc.alerter.Alert(ctx, Alert{Source: source, At: sc.sync.clock.Now(), Err: err})
	}
	return err
}

func (sc *Scheduler) planned(source domain.Source) bool {
	return slices.ContainsFunc(sc.plans, func(p Plan) bool { return p.Source == source })
}
