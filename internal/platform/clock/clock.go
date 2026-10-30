// SPDX-License-Identifier: AGPL-3.0-only

// Package clock gives the commands the system time, pauses and jitter that
// the use cases take as ports, so the use cases stay testable.
package clock

import (
	"context"
	"math/rand/v2"
	"time"
)

// System is the system clock.
type System struct{}

// Now returns the current time.
func (System) Now() time.Time {
	return time.Now()
}

// Sleep returns after d, or with the error of ctx once it is done.
func (System) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Jitter returns a random duration in [0, bound), or 0 when bound is not
// positive.
func Jitter(bound time.Duration) time.Duration {
	if bound <= 0 {
		return 0
	}
	//nolint:gosec // G404: jitter spreads syncs, it needs no cryptographic randomness.
	return rand.N(bound)
}
