// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"fmt"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// KeyPurgeEvery is how often the API deletes the records of the keys
// revoked longer than domain.RevokedKeyRetention ago.
const KeyPurgeEvery = 24 * time.Hour

// KeyPurger deletes the records of revoked API keys.
type KeyPurger interface {
	// PurgeRevokedKeys deletes the keys revoked before cutoff and returns
	// how many it deleted.
	PurgeRevokedKeys(ctx context.Context, cutoff time.Time) (int64, error)
}

// PurgeReport receives the outcome of each purge: the number of keys
// deleted, or why the purge failed.
type PurgeReport func(ctx context.Context, deleted int64, err error)

// KeyPurge deletes, at an interval, the records of the API keys revoked
// longer than domain.RevokedKeyRetention ago (ADR 0006).
type KeyPurge struct {
	keys    KeyPurger
	clock   Clock
	sleeper Sleeper
	every   time.Duration
}

// NewKeyPurge returns a KeyPurge of keys, purging every interval.
func NewKeyPurge(keys KeyPurger, clock Clock, sleeper Sleeper, every time.Duration) *KeyPurge {
	return &KeyPurge{keys: keys, clock: clock, sleeper: sleeper, every: every}
}

// Once deletes the keys revoked longer than domain.RevokedKeyRetention ago
// and returns how many it deleted.
func (p *KeyPurge) Once(ctx context.Context) (int64, error) {
	cutoff := p.clock.Now().UTC().Add(-domain.RevokedKeyRetention)
	n, err := p.keys.PurgeRevokedKeys(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("purge API keys revoked before %s: %w", cutoff.Format(time.RFC3339), err)
	}
	return n, nil
}

// Run purges at once, then at each interval, until ctx is done, and hands
// the outcome of each purge to report. A failed purge is tried again at
// the next interval.
func (p *KeyPurge) Run(ctx context.Context, report PurgeReport) {
	for {
		n, err := p.Once(ctx)
		if ctx.Err() != nil {
			return
		}
		report(ctx, n, err)
		if p.sleeper.Sleep(ctx, p.every) != nil {
			return
		}
	}
}
