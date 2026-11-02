// SPDX-License-Identifier: AGPL-3.0-only

package listing

import (
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// NumberingReader is the app.NumberingReader of a numbering plan. It
// indexes each snapshot in service once, at the first read after it is
// swapped in, and is safe for concurrent use.
type NumberingReader struct {
	cache cache[*domain.NumberingPlan]
}

// NewNumberingReader returns the NumberingReader of the plan of source in
// snapshots, which build indexes.
func NewNumberingReader(snapshots Snapshots, source domain.Source, build func(domain.Snapshot) (*domain.NumberingPlan, error)) *NumberingReader {
	return &NumberingReader{cache: cache[*domain.NumberingPlan]{snapshots: snapshots, source: source, build: build}}
}

// Numbering returns the plan in service and when it was fetched,
// app.ErrNoSnapshot when none is, or the error of its build.
func (r *NumberingReader) Numbering() (*domain.NumberingPlan, time.Time, error) {
	return r.cache.get()
}
