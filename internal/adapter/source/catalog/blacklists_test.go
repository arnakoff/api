// SPDX-License-Identifier: AGPL-3.0-only

package catalog_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// noSnapshot serves no snapshot.
type noSnapshot struct{}

func (noSnapshot) Snapshot(domain.Source) (domain.Snapshot, bool) { return domain.Snapshot{}, false }

func TestBlacklistsTheConfigurationTurnsOn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts catalog.Options
		want []domain.Source
	}{
		{"default", catalog.Options{}, []domain.Source{"AMF"}},
		{"with ACPR", catalog.Options{ACPR: true}, []domain.Source{"AMF", "ACPR"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []domain.Source
			for _, b := range catalog.Blacklists(tt.opts, noSnapshot{}) {
				got = append(got, b.Rule.Source())
				// Each reader reads its rule's source: none is stored here.
				if _, _, err := b.Reader.Blacklist(); !errors.Is(err, app.ErrNoSnapshot) {
					t.Errorf("%s reader error = %v, want ErrNoSnapshot", b.Rule.Source(), err)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("blacklists = %v, want %v", got, tt.want)
			}
		})
	}
}
