// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/arnakoff/api/internal/domain"
)

// defaultBatchLimit is how many inputs of a batch CheckAll checks at once
// unless WithBatchLimit says otherwise. Each check already consults its
// registries concurrently: the limit bounds the load one batch puts on them.
const defaultBatchLimit = 8

// WithBatchLimit makes CheckAll check at most n inputs at once; n below 1
// means one.
func WithBatchLimit(n int) CheckOption {
	return func(c *Check) { c.batchLimit = max(n, 1) }
}

// CheckAll returns the report of each input, in the order of ins, as Check
// returns it under the batch deadline. It checks several inputs at once, up to the batch limit, and
// the timeout bounds the whole batch rather than each check: an input whose
// turn comes at the deadline is reported with its registries unavailable.
func (c *Check) CheckAll(ctx context.Context, ins []domain.Input) []domain.Report {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	reports := make([]domain.Report, len(ins))
	var g errgroup.Group
	g.SetLimit(c.batchLimit)
	for i, in := range ins {
		g.Go(func() error {
			reports[i] = c.check(ctx, in)
			return nil
		})
	}
	// A check never fails: its sources are reported unavailable instead.
	_ = g.Wait()
	return reports
}
