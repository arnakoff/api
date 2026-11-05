// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// KeyPurgeEvery is how often the API deletes the records of the keys
// revoked longer than domain.RevokedKeyRetention ago, and the login
// requests past domain.LoginRequestRetention.
const KeyPurgeEvery = 24 * time.Hour

// KeyPurger deletes the records of revoked API keys and of login requests.
type KeyPurger interface {
	// PurgeRevokedKeys deletes the keys revoked before cutoff and returns
	// how many it deleted.
	PurgeRevokedKeys(ctx context.Context, cutoff time.Time) (int64, error)
	// PurgeLoginRequests deletes the login requests whose link was used,
	// or else expired, before cutoff and returns how many it deleted.
	PurgeLoginRequests(ctx context.Context, cutoff time.Time) (int64, error)
}

// PurgeCount is what a purge, or the erasure of an account, deleted.
type PurgeCount struct {
	Keys          int64
	LoginRequests int64
}

// PurgeReport receives the outcome of each purge: what it deleted, and
// why a part of it failed.
type PurgeReport func(ctx context.Context, deleted PurgeCount, err error)

// KeyPurge deletes, at an interval, the records of the API keys revoked
// longer than domain.RevokedKeyRetention ago and the login requests past
// domain.LoginRequestRetention (ADR 0006).
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
// and the login requests past domain.LoginRequestRetention, and returns
// how many of each it deleted. A part that fails does not stop the other.
func (p *KeyPurge) Once(ctx context.Context) (PurgeCount, error) {
	now := p.clock.Now().UTC()
	keys, keyErr := purgeBefore(ctx, "API keys revoked", now.Add(-domain.RevokedKeyRetention), p.keys.PurgeRevokedKeys)
	logins, loginErr := purgeBefore(ctx, "login requests ended", now.Add(-domain.LoginRequestRetention), p.keys.PurgeLoginRequests)
	return PurgeCount{Keys: keys, LoginRequests: logins}, errors.Join(keyErr, loginErr)
}

// purgeBefore runs purge on what ended before cutoff, and names what in
// its error.
func purgeBefore(ctx context.Context, what string, cutoff time.Time, purge func(context.Context, time.Time) (int64, error)) (int64, error) {
	n, err := purge(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("purge %s before %s: %w", what, cutoff.Format(time.RFC3339), err)
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
