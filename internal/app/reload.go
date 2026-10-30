// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"fmt"
	"time"
)

// Reloader restores, at an interval, the snapshots the sync process stored,
// so that the API process serves them without a restart. It alerts when a
// source starts failing to reload, not again until it has recovered.
type Reloader struct {
	sync    *Sync
	sleeper Sleeper
	alerter Alerter
	every   time.Duration
}

// NewReloader returns a Reloader of the sources of s, checking every
// interval.
func NewReloader(s *Sync, sleeper Sleeper, alerter Alerter, every time.Duration) *Reloader {
	return &Reloader{sync: s, sleeper: sleeper, alerter: alerter, every: every}
}

// Run reloads every source at each interval until ctx is done. Restore the
// snapshots once before, so the first ones are in service at once.
func (r *Reloader) Run(ctx context.Context) {
	failing := make([]bool, len(r.sync.feeds))
	for r.sleeper.Sleep(ctx, r.every) == nil {
		for i, f := range r.sync.feeds {
			err := r.sync.restore(ctx, f)
			if err != nil && !failing[i] && ctx.Err() == nil {
				err = fmt.Errorf("reload %s: %w", f.spec.Source, err)
				r.alerter.Alert(ctx, Alert{Source: f.spec.Source, At: r.sync.clock.Now(), Err: err})
			}
			failing[i] = err != nil
		}
	}
}
