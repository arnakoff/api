// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"fmt"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Plan indexes the ranges of a numbering snapshot. It rejects with
// ErrFormat a snapshot whose rows do not read as ranges or whose ranges
// overlap: a number would then have two holders.
func Plan(snap domain.Snapshot) (*domain.NumberingPlan, error) {
	if snap.Source != NumberingSource || !slices.Equal(snap.Header, numberingColumns) {
		return nil, fmt.Errorf("%w: not a %s numbering snapshot", ErrFormat, NumberingSource)
	}
	ranges := make([]domain.NumberRange, 0, len(snap.Rows))
	for i, row := range snap.Rows {
		r, err := rangeOf(row)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		ranges = append(ranges, r)
	}
	plan, err := domain.NewNumberingPlan(ranges)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFormat, err)
	}
	return plan, nil
}

// rangeOf reads a snapshot row, laid out as numberingColumns.
func rangeOf(row []string) (domain.NumberRange, error) {
	if len(row) != len(numberingColumns) {
		return domain.NumberRange{}, fmt.Errorf("%w: %d fields", ErrFormat, len(row))
	}
	assigned, err := time.Parse(time.DateOnly, row[7])
	if err != nil {
		return domain.NumberRange{}, fmt.Errorf("%w: %s not AAAA-MM-JJ", ErrFormat, ColumnAssignedOn)
	}
	r, err := domain.NewNumberRange(row[1], row[2], domain.NumberAssignment{
		Prefix: row[0], Operator: row[3], OperatorName: row[4], OperatorSIREN: row[5],
		Territory: row[6], AssignedOn: assigned,
	})
	if err != nil {
		return domain.NumberRange{}, fmt.Errorf("%w: %w", ErrFormat, err)
	}
	return r, nil
}
