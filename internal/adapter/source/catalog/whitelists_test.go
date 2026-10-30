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

func TestWhitelists(t *testing.T) {
	t.Parallel()

	var got []domain.Source
	for _, w := range catalog.Whitelists(noSnapshot{}) {
		got = append(got, w.Rule.Source())
		// Each reader reads its rule's source: none is stored here.
		if _, _, err := w.Reader.Whitelist(); !errors.Is(err, app.ErrNoSnapshot) {
			t.Errorf("%s reader error = %v, want ErrNoSnapshot", w.Rule.Source(), err)
		}
	}
	if want := []domain.Source{"AMF PSAN", "AMF biens divers"}; !slices.Equal(got, want) {
		t.Errorf("whitelists = %v, want %v", got, want)
	}
}
