// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Sync errors, checked with errors.Is.
var (
	ErrUnknownSource   = errors.New("unknown source")
	ErrDuplicateSource = errors.New("duplicate source")
)

// Sync downloads each source, validates the new snapshot against the
// connector's spec and the last valid one, stores it, then swaps it in for
// the readers. A rejected snapshot leaves the last valid one in service.
// It is safe for concurrent use.
type Sync struct {
	store SnapshotStore
	clock Clock
	feeds []*feed // in declaration order
}

// NewSync returns a Sync of the sources the fetchers declare, or
// ErrDuplicateSource when two declare the same one.
func NewSync(store SnapshotStore, clock Clock, fetchers ...SourceFetcher) (*Sync, error) {
	s := &Sync{store: store, clock: clock}
	for _, f := range fetchers {
		spec := f.Spec()
		if s.feed(spec.Source) != nil {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateSource, spec.Source)
		}
		s.feeds = append(s.feeds, &feed{fetcher: f, spec: spec})
	}
	return s, nil
}

// Run syncs every source in turn and joins their errors: one source failing
// does not stop the others.
func (s *Sync) Run(ctx context.Context) error {
	var errs []error
	for _, f := range s.feeds {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}
		errs = append(errs, s.run(ctx, f))
	}
	return errors.Join(errs...)
}

// RunSource syncs one source, or returns ErrUnknownSource.
func (s *Sync) RunSource(ctx context.Context, source domain.Source) error {
	f := s.feed(source)
	if f == nil {
		return fmt.Errorf("%w: %s", ErrUnknownSource, source)
	}
	return s.run(ctx, f)
}

// Snapshot returns the snapshot in service for source, if any. It is shared
// with every reader: treat it as read-only.
func (s *Sync) Snapshot(source domain.Source) (domain.Snapshot, bool) {
	if f := s.feed(source); f != nil {
		if snap := f.current.Load(); snap != nil {
			return *snap, true
		}
	}
	return domain.Snapshot{}, false
}

// States returns the sync state of every source, in declaration order.
func (s *Sync) States() []SourceSync {
	states := make([]SourceSync, len(s.feeds))
	for i, f := range s.feeds {
		states[i] = f.sourceSync()
	}
	return states
}

// run fetches, validates, stores and swaps in a new snapshot of f, and
// records the outcome.
func (s *Sync) run(ctx context.Context, f *feed) error {
	f.run.Lock()
	defer f.run.Unlock()
	now := s.clock.Now()
	if err := s.refresh(ctx, f, now); err != nil {
		err = fmt.Errorf("sync %s: %w", f.spec.Source, err)
		f.failure.Store(&failure{at: now, err: err})
		return err
	}
	return nil
}

func (s *Sync) refresh(ctx context.Context, f *feed, now time.Time) error {
	snap, err := f.fetcher.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	snap.FetchedAt = now
	if err := f.spec.Validate(snap, f.rows()); err != nil {
		return err
	}
	if err := s.store.Save(ctx, snap); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	f.current.Store(&snap)
	return nil
}

// feed returns the feed of source, or nil.
func (s *Sync) feed(source domain.Source) *feed {
	for _, f := range s.feeds {
		if f.spec.Source == source {
			return f
		}
	}
	return nil
}
