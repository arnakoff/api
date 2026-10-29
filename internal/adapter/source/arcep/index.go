// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Assignment is the range a phone number falls in and who it is assigned
// to.
type Assignment struct {
	// Prefix is the range's prefix ("EZABPQM"), such as "0612".
	Prefix string
	// Operator is the code of the operator holding the range, such as
	// "SFR0".
	Operator string
	// OperatorName and OperatorSIREN are "" for an operator missing from
	// the operators file; OperatorSIREN also for a foreign operator.
	OperatorName  string
	OperatorSIREN string
	// Territory is where the range is used, as ARCEP names it
	// ("Métropole", "Martinique", "National").
	Territory string
	// AssignedOn is the date the range was assigned.
	AssignedOn time.Time
}

// numberRange is an assigned range of numbers of the same number of
// digits, first and last included.
type numberRange struct {
	digits      int
	first, last uint64
	assignment  Assignment
}

// compareStart orders ranges by number of digits, then by first number.
func compareStart(a, b numberRange) int {
	return cmp.Or(cmp.Compare(a.digits, b.digits), cmp.Compare(a.first, b.first))
}

// Index looks phone numbers up among the ranges of a numbering snapshot.
// It is read-only once built: readers share it.
type Index struct {
	ranges []numberRange // sorted by compareStart, none overlapping
}

// NewIndex indexes the ranges of a numbering snapshot. It rejects with
// ErrFormat a snapshot whose rows do not read as ranges or whose ranges
// overlap: a number would then have two holders.
func NewIndex(snap domain.Snapshot) (*Index, error) {
	if snap.Source != NumberingSource || !slices.Equal(snap.Header, numberingColumns) {
		return nil, fmt.Errorf("%w: not a %s numbering snapshot", ErrFormat, NumberingSource)
	}
	ranges := make([]numberRange, 0, len(snap.Rows))
	for i, row := range snap.Rows {
		r, err := rangeOf(row)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		ranges = append(ranges, r)
	}
	slices.SortFunc(ranges, compareStart)
	for i := 1; i < len(ranges); i++ {
		if prev := ranges[i-1]; prev.digits == ranges[i].digits && ranges[i].first <= prev.last {
			return nil, fmt.Errorf("%w: ranges overlap", ErrFormat)
		}
	}
	return &Index{ranges: ranges}, nil
}

// rangeOf reads a snapshot row, laid out as numberingColumns.
func rangeOf(row []string) (numberRange, error) {
	if len(row) != len(numberingColumns) {
		return numberRange{}, fmt.Errorf("%w: %d fields", ErrFormat, len(row))
	}
	first, last, err := parseBounds(row[1], row[2])
	if err != nil {
		return numberRange{}, err
	}
	assigned, err := time.Parse(time.DateOnly, row[7])
	if err != nil {
		return numberRange{}, fmt.Errorf("%w: %s not AAAA-MM-JJ", ErrFormat, ColumnAssignedOn)
	}
	return numberRange{
		digits: len(row[1]), first: first, last: last,
		assignment: Assignment{
			Prefix: row[0], Operator: row[3], OperatorName: row[4], OperatorSIREN: row[5],
			Territory: row[6], AssignedOn: assigned,
		},
	}, nil
}

// Lengths of the numbers of the French numbering plan: a national number
// starts with the trunk prefix 0 and has ten digits, or fourteen for the
// 070 numbers of connected objects; a short number has at most six.
const (
	minNationalDigits = 10
	maxShortDigits    = 6
)

// parseBounds reads the first and last numbers of a range, written with
// the same number of digits, first not after last. A length outside the
// plan, as when a spreadsheet drops the leading zero, is rejected.
func parseBounds(first, last string) (uint64, uint64, error) {
	f, errF := parseDigits(first)
	l, errL := parseDigits(last)
	switch {
	case errF != nil || errL != nil:
		return 0, 0, fmt.Errorf("%w: range bound not a number", ErrFormat)
	case len(first) != len(last):
		return 0, 0, fmt.Errorf("%w: range bounds of %d and %d digits", ErrFormat, len(first), len(last))
	case first[0] == '0' && len(first) < minNationalDigits, first[0] != '0' && len(first) > maxShortDigits:
		return 0, 0, fmt.Errorf("%w: range of %d digits", ErrFormat, len(first))
	case f > l:
		return 0, 0, fmt.Errorf("%w: range first after last", ErrFormat)
	}
	return f, l, nil
}

// parseDigits reads a non-empty string of ASCII digits.
func parseDigits(s string) (uint64, error) {
	if s == "" || s[0] < '0' || s[0] > '9' {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseUint(s, 10, 64)
}

// Lookup returns the assignment of the range p falls in. It reports false
// for a number outside the French numbering plan or in no assigned range:
// such a number cannot be in service.
func (ix *Index) Lookup(p domain.Phone) (Assignment, bool) {
	national, ok := p.FrenchNational()
	if !ok {
		return Assignment{}, false
	}
	n, err := strconv.ParseUint(national, 10, 64)
	if err != nil {
		return Assignment{}, false
	}
	target := numberRange{digits: len(national), first: n}
	i, found := slices.BinarySearchFunc(ix.ranges, target, compareStart)
	if !found {
		// ranges[i-1] is the last range starting before n.
		i--
	}
	if i < 0 || ix.ranges[i].digits != target.digits || n > ix.ranges[i].last {
		return Assignment{}, false
	}
	return ix.ranges[i].assignment, true
}
