// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// Snapshot is one validated copy of an upstream file, as rows of fields
// under a header. It is read-only once built: readers share it.
type Snapshot struct {
	// Source is the upstream source the file comes from.
	Source Source
	// Header names the fields of each row, in order.
	Header []string
	// Rows holds the entries, one slice of fields per row.
	Rows [][]string
	// FetchedAt is when the file was downloaded.
	FetchedAt time.Time
	// Newest is the date of the most recent entry, zero when the source
	// dates none.
	Newest time.Time
}

// Snapshot validation errors, checked with errors.Is.
var (
	ErrSnapshotSource        = errors.New("snapshot of another source")
	ErrSnapshotEmpty         = errors.New("snapshot has no row")
	ErrSnapshotTooSmall      = errors.New("snapshot has fewer rows than expected")
	ErrSnapshotColumnMissing = errors.New("missing column")
	ErrSnapshotShrunk        = errors.New("shrunk by more than 20 %")
)

// maxShrinkPercent is how much a snapshot may shrink from the previous one
// before it is rejected (CLAUDE.md, fail safe on sync).
const maxShrinkPercent = 20

// SnapshotSpec is what a connector declares about its source's file: the
// columns it reads and the least number of rows a sane file holds.
type SnapshotSpec struct {
	Source  Source
	Columns []string
	MinRows int
}

// Validate reports why s cannot replace a snapshot of previousRows rows
// (0 when there is none), or nil when it can.
func (spec SnapshotSpec) Validate(s Snapshot, previousRows int) error {
	if err := spec.validate(s, previousRows); err != nil {
		return fmt.Errorf("snapshot %s: %w", spec.Source, err)
	}
	return nil
}

func (spec SnapshotSpec) validate(s Snapshot, previousRows int) error {
	if s.Source != spec.Source {
		return fmt.Errorf("%w: %s", ErrSnapshotSource, s.Source)
	}
	if len(s.Rows) == 0 {
		return ErrSnapshotEmpty
	}
	if len(s.Rows) < spec.MinRows {
		return fmt.Errorf("%w: %d rows, %d at least", ErrSnapshotTooSmall, len(s.Rows), spec.MinRows)
	}
	for _, c := range spec.Columns {
		if !slices.Contains(s.Header, c) {
			return fmt.Errorf("%w: %q", ErrSnapshotColumnMissing, c)
		}
	}
	if len(s.Rows)*100 < previousRows*(100-maxShrinkPercent) {
		return fmt.Errorf("%w: %d rows, %d before", ErrSnapshotShrunk, len(s.Rows), previousRows)
	}
	return nil
}
