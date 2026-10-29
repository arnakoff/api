// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
)

// Restore puts in service the stored snapshot of every source when it is
// valid and newer than the one in service. It runs at startup, and in the
// API process to pick up what the sync process stored. A source with no
// stored snapshot is skipped; errors of the others are joined.
func (s *Sync) Restore(ctx context.Context) error {
	var errs []error
	for _, f := range s.feeds {
		if err := s.restore(ctx, f); err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", f.spec.Source, err))
		}
	}
	return errors.Join(errs...)
}

func (s *Sync) restore(ctx context.Context, f *feed) error {
	f.run.Lock()
	defer f.run.Unlock()
	snap, err := s.store.Load(ctx, f.spec.Source)
	if errors.Is(err, ErrNoSnapshot) {
		return nil
	}
	if err != nil {
		return err
	}
	if cur := f.current.Load(); cur != nil && !snap.FetchedAt.After(cur.FetchedAt) {
		return nil
	}
	if err := f.spec.Validate(snap, 0); err != nil {
		return err
	}
	f.current.Store(&snap)
	return nil
}
