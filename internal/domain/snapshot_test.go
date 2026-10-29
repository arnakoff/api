// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

// rows returns n rows of two fields.
func rows(n int) [][]string {
	out := make([][]string, n)
	for i := range out {
		out[i] = []string{"site-" + strconv.Itoa(i) + ".example", "2026-10-29"}
	}
	return out
}

func snapshotOf(n int) domain.Snapshot {
	return domain.Snapshot{
		Source: "AMF",
		Header: []string{"nom", "date_inscription"},
		Rows:   rows(n),
	}
}

func TestSnapshotSpec_Validate(t *testing.T) {
	t.Parallel()

	spec := domain.SnapshotSpec{
		Source:  "AMF",
		Columns: []string{"nom", "date_inscription"},
		MinRows: 10,
	}
	tests := []struct {
		name     string
		snapshot domain.Snapshot
		previous int // rows of the previous snapshot, 0 for none
		want     error
	}{
		{"first snapshot", snapshotOf(10), 0, nil},
		{"same size", snapshotOf(100), 100, nil},
		{"grows", snapshotOf(150), 100, nil},
		{"shrinks by exactly 20 %", snapshotOf(80), 100, nil},
		{"empty", snapshotOf(0), 0, domain.ErrSnapshotEmpty},
		{"empty with a previous one", snapshotOf(0), 100, domain.ErrSnapshotEmpty},
		{"under the minimum", snapshotOf(9), 0, domain.ErrSnapshotTooSmall},
		{"shrinks by 21 %", snapshotOf(79), 100, domain.ErrSnapshotShrunk},
		{"shrinks by 30 %", snapshotOf(70), 100, domain.ErrSnapshotShrunk},
		{
			"missing column",
			domain.Snapshot{Source: "AMF", Header: []string{"nom"}, Rows: rows(10)},
			0, domain.ErrSnapshotColumnMissing,
		},
		{
			"other source",
			domain.Snapshot{Source: "ACPR", Header: spec.Columns, Rows: rows(10)},
			0, domain.ErrSnapshotSource,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := spec.Validate(tt.snapshot, tt.previous)
			if !errors.Is(err, tt.want) {
				t.Errorf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSnapshotSpec_ValidateNamesTheMissingColumn(t *testing.T) {
	t.Parallel()

	spec := domain.SnapshotSpec{Source: "AMF", Columns: []string{"nom", "categorie"}}
	err := spec.Validate(snapshotOf(1), 0)
	if err == nil || err.Error() != `snapshot AMF: missing column: "categorie"` {
		t.Errorf("Validate() = %v, want the missing column named", err)
	}
}

func TestSnapshotSpec_ValidateGivesTheShrinkFigures(t *testing.T) {
	t.Parallel()

	spec := domain.SnapshotSpec{Source: "AMF", Columns: []string{"nom"}}
	err := spec.Validate(snapshotOf(70), 100)
	if err == nil || err.Error() != "snapshot AMF: shrunk by more than 20 %: 70 rows, 100 before" {
		t.Errorf("Validate() = %v, want the row counts", err)
	}
}
