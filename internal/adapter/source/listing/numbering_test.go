// SPDX-License-Identifier: AGPL-3.0-only

package listing_test

import (
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestNumberingReader(t *testing.T) {
	t.Parallel()

	var s snapshots
	errBuild := errors.New("bad plan")
	builds := 0
	build := func(snap domain.Snapshot) (*domain.NumberingPlan, error) {
		builds++
		if len(snap.Rows) == 0 {
			return nil, errBuild
		}
		return domain.NewNumberingPlan(nil)
	}
	r := listing.NewNumberingReader(&s, "ARCEP", build)
	if _, _, err := r.Numbering(); !errors.Is(err, app.ErrNoSnapshot) {
		t.Fatalf("Numbering() without snapshot error = %v, want ErrNoSnapshot", err)
	}
	s.put(domain.Snapshot{Source: "ARCEP", FetchedAt: fetchedAt, Rows: [][]string{{"row"}}})
	plan, at, err := r.Numbering()
	if err != nil || plan == nil || !at.Equal(fetchedAt) {
		t.Fatalf("Numbering() = %v, %v, %v; want a plan, %v, nil", plan, at, err, fetchedAt)
	}
	if _, _, err := r.Numbering(); err != nil || builds != 1 {
		t.Errorf("second Numbering() error = %v after %d builds, want nil after 1", err, builds)
	}
	s.put(domain.Snapshot{Source: "ARCEP", FetchedAt: fetchedAt.Add(1)})
	if _, _, err := r.Numbering(); !errors.Is(err, errBuild) {
		t.Errorf("Numbering() of a bad snapshot error = %v, want %v", err, errBuild)
	}
}
