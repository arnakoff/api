// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// NumberingSource is the name of the numbering plan, as shown to users.
const NumberingSource domain.Source = "ARCEP"

// NumberingURL is the file of the assigned numbering ranges (MAJNUM).
const NumberingURL = "https://extranet.arcep.fr/uploads/MAJNUM.csv"

// Columns of a numbering snapshot row.
const (
	// ColumnPrefix is the range's prefix ("EZABPQM").
	ColumnPrefix = "prefix"
	// ColumnFirst and ColumnLast are the first and last numbers of the
	// range, in national form, of the same number of digits.
	ColumnFirst = "first"
	ColumnLast  = "last"
	// ColumnOperator is the code of the operator holding the range.
	ColumnOperator = "operator"
	// ColumnOperatorName and ColumnOperatorSIREN are the operator's name
	// and SIREN, "" when the operators file does not give them.
	ColumnOperatorName  = "operator_name"
	ColumnOperatorSIREN = "operator_siren"
	// ColumnTerritory is where the range is used.
	ColumnTerritory = "territory"
	// ColumnAssignedOn is the assignment date, AAAA-MM-JJ.
	ColumnAssignedOn = "assigned_on"
)

var numberingColumns = []string{
	ColumnPrefix, ColumnFirst, ColumnLast, ColumnOperator,
	ColumnOperatorName, ColumnOperatorSIREN, ColumnTerritory, ColumnAssignedOn,
}

// numberingHeader is the header of the published file (docs/sources.md).
var numberingHeader = []string{"EZABPQM", "Tranche_Debut", "Tranche_Fin", "Mnémo", "Territoire", "Date_Attribution"}

// numberingMinRows is the least number of rows a sane file gives. On
// 29/10/2026 the file gave 20 931; ranges are returned and reassigned, and
// the margin allows for a wave of returns.
const numberingMinRows = 18000

// ParseNumbering reads the published numbering file into a snapshot, one
// row per assigned range, with the name and SIREN of its holder from
// operators. A holder missing from operators, which ARCEP updates on
// another schedule, is kept with its code only. A file that departs from
// the recorded format, or whose ranges overlap, is rejected with ErrFormat
// rather than half read.
func ParseNumbering(r io.Reader, operators map[string]Operator) (domain.Snapshot, error) {
	cr, err := newReader(r, numberingHeader)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snap := domain.Snapshot{Source: NumberingSource, Header: numberingColumns}
	err = cr.Each(func(rec []string) error { return addRange(&snap, rec, operators) })
	if err != nil {
		return domain.Snapshot{}, err
	}
	if _, err := NewIndex(snap); err != nil {
		return domain.Snapshot{}, err
	}
	return snap, nil
}

// addRange appends the range rec to snap and moves snap.Newest to its
// assignment date.
func addRange(snap *domain.Snapshot, rec []string, operators map[string]Operator) error {
	prefix, first, last, code, territory := rec[0], rec[1], rec[2], rec[3], rec[4]
	if prefix == "" {
		return errEmpty(numberingHeader[0])
	}
	if _, _, err := parseBounds(first, last); err != nil {
		return err
	}
	if !strings.HasPrefix(first, prefix) || !strings.HasPrefix(last, prefix) {
		return fmt.Errorf("%w: range outside its prefix", ErrFormat)
	}
	if code == "" {
		return errEmpty(numberingHeader[3])
	}
	if territory == "" {
		return errEmpty(numberingHeader[4])
	}
	assigned, err := parseFrenchDate(numberingHeader[5], rec[5])
	if err != nil {
		return err
	}
	op := operators[code]
	snap.Rows = append(snap.Rows, []string{
		prefix, first, last, code, op.Name, op.SIREN, territory, assigned.Format(time.DateOnly),
	})
	if assigned.After(snap.Newest) {
		snap.Newest = assigned
	}
	return nil
}
