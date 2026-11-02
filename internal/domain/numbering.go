// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"
)

// ErrInvalidRange means a range of phone numbers is not one of the French
// numbering plan, or overlaps another.
var ErrInvalidRange = errors.New("invalid numbering range")

// Lengths of the numbers of the French numbering plan: a national number
// starts with the trunk prefix 0 and has ten digits, or fourteen for the
// 070 numbers of connected objects; a short number has at most six.
const (
	minNationalDigits = 10
	maxShortDigits    = 6
)

// NumberAssignment is a range of phone numbers the national regulator
// assigned and who it assigned it to. It says which operator holds a
// number's range, nothing about who uses the number, which may have been
// ported to another operator.
type NumberAssignment struct {
	// Prefix is the range's prefix ("EZABPQM"), such as "0612".
	Prefix string
	// Operator is the code of the operator holding the range, such as
	// "SFR0".
	Operator string
	// OperatorName and OperatorSIREN are "" when the regulator does not
	// give them; OperatorSIREN also for a foreign operator.
	OperatorName  string
	OperatorSIREN string
	// Territory is where the range is used, as the regulator names it
	// ("Métropole", "Martinique", "National").
	Territory string
	// AssignedOn is the day the range was assigned.
	AssignedOn time.Time
}

// NumberRange is an assigned range of numbers of the same number of
// digits, first and last included.
type NumberRange struct {
	digits      int
	first, last uint64
	assignment  NumberAssignment
}

// NewNumberRange returns the range from first to last, numbers in national
// form of the same number of digits, assigned as a. It rejects with
// ErrInvalidRange bounds that are not numbers of the plan, as when a
// spreadsheet drops the leading zero, or a first number after the last.
func NewNumberRange(first, last string, a NumberAssignment) (NumberRange, error) {
	f, errF := parseDigits(first)
	l, errL := parseDigits(last)
	switch {
	case errF != nil || errL != nil:
		return NumberRange{}, fmt.Errorf("%w: bound not a number", ErrInvalidRange)
	case len(first) != len(last):
		return NumberRange{}, fmt.Errorf("%w: bounds of %d and %d digits", ErrInvalidRange, len(first), len(last))
	case first[0] == '0' && len(first) < minNationalDigits, first[0] != '0' && len(first) > maxShortDigits:
		return NumberRange{}, fmt.Errorf("%w: %d digits", ErrInvalidRange, len(first))
	case f > l:
		return NumberRange{}, fmt.Errorf("%w: first after last", ErrInvalidRange)
	}
	return NumberRange{digits: len(first), first: f, last: l, assignment: a}, nil
}

// parseDigits reads a non-empty string of ASCII digits.
func parseDigits(s string) (uint64, error) {
	if s == "" || s[0] < '0' || s[0] > '9' {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseUint(s, 10, 64)
}

// compareStart orders ranges by number of digits, then by first number.
func compareStart(a, b NumberRange) int {
	return cmp.Or(cmp.Compare(a.digits, b.digits), cmp.Compare(a.first, b.first))
}

// NumberingPlan indexes the assigned ranges of the French numbering plan.
// Once built it is read-only: readers share it.
type NumberingPlan struct {
	ranges []NumberRange // sorted by compareStart, none overlapping
}

// NewNumberingPlan indexes ranges, which it does not change. It rejects
// with ErrInvalidRange ranges that overlap: a number would then have two
// holders.
func NewNumberingPlan(ranges []NumberRange) (*NumberingPlan, error) {
	sorted := slices.Clone(ranges)
	slices.SortFunc(sorted, compareStart)
	for i := 1; i < len(sorted); i++ {
		if prev := sorted[i-1]; prev.digits == sorted[i].digits && sorted[i].first <= prev.last {
			return nil, fmt.Errorf("%w: ranges overlap", ErrInvalidRange)
		}
	}
	return &NumberingPlan{ranges: sorted}, nil
}

// NumberingPlanCovers reports whether p belongs to the French numbering
// plan, metropolitan France and the overseas departments and regions that
// dial as French numbers: only those can be looked up in a NumberingPlan.
func NumberingPlanCovers(p Phone) bool {
	_, ok := p.FrenchNational()
	return ok
}

// Lookup returns the assignment of the range p falls in. It reports false
// for a number outside the French numbering plan or in no assigned range:
// such a number cannot be in service.
func (np *NumberingPlan) Lookup(p Phone) (NumberAssignment, bool) {
	national, ok := p.FrenchNational()
	if !ok {
		return NumberAssignment{}, false
	}
	n, err := strconv.ParseUint(national, 10, 64)
	if err != nil {
		return NumberAssignment{}, false
	}
	target := NumberRange{digits: len(national), first: n}
	i, found := slices.BinarySearchFunc(np.ranges, target, compareStart)
	if !found {
		// ranges[i-1] is the last range starting before n.
		i--
	}
	if i < 0 || np.ranges[i].digits != target.digits || n > np.ranges[i].last {
		return NumberAssignment{}, false
	}
	return np.ranges[i].assignment, true
}
