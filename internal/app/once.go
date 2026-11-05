// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"sync/atomic"
)

// Once runs one step of administration, such as the migrations (#259) or a
// restore (#297), and tells whether it succeeded, for the readiness probe
// of the private container that runs it.
type Once struct {
	step func(ctx context.Context) error
	done atomic.Bool
}

// NewOnce returns a Once of step, not yet run.
func NewOnce(step func(ctx context.Context) error) *Once {
	return &Once{step: step}
}

// Run runs the step and returns its error. Ready reports true once it
// succeeded.
func (o *Once) Run(ctx context.Context) error {
	if err := o.step(ctx); err != nil {
		return err
	}
	o.done.Store(true)
	return nil
}

// Ready reports whether Run succeeded.
func (o *Once) Ready() bool {
	return o.done.Load()
}
