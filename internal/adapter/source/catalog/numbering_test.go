// SPDX-License-Identifier: AGPL-3.0-only

package catalog_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/source/arcep"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/domain"
)

// arcepSnapshot serves an ARCEP snapshot of one range of the fixture.
type arcepSnapshot struct{}

func (arcepSnapshot) Snapshot(source domain.Source) (domain.Snapshot, bool) {
	if source != arcep.NumberingSource {
		return domain.Snapshot{}, false
	}
	return domain.Snapshot{
		Source:    arcep.NumberingSource,
		Header:    arcep.NewNumbering(nil, "", "").Spec().Columns,
		Rows:      [][]string{{"0140", "0140000000", "0140999999", "FRTE", "Orange", "380129866", "Métropole", "2017-01-01"}},
		FetchedAt: time.Date(2026, time.October, 30, 8, 0, 0, 0, time.UTC),
	}, true
}

func TestNumberingReadsTheARCEPSnapshot(t *testing.T) {
	t.Parallel()

	plan, _, err := catalog.Numbering(arcepSnapshot{})[0].Reader.Numbering()
	if err != nil {
		t.Fatalf("Numbering() error = %v", err)
	}
	p, err := domain.ParsePhone("01 40 12 34 56")
	if err != nil {
		t.Fatal(err)
	}
	if a, found := plan.Lookup(p); !found || a.Operator != "FRTE" {
		t.Errorf("Lookup() = %+v, %v, want the FRTE range", a, found)
	}
}
