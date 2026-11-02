// SPDX-License-Identifier: AGPL-3.0-only

package catalog_test

import (
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/app"
)

func TestCatalogDescribesEverySourceItSyncs(t *testing.T) {
	t.Parallel()
	for _, opts := range []catalog.Options{{}, {ACPR: true}} {
		c := catalog.New(catalog.Offline{}, opts)
		s, err := app.NewSync(nil, nil, c.Fetchers()...)
		if err != nil {
			t.Fatalf("NewSync: %v", err)
		}
		if _, err := app.NewSourceList(s, c.Infos()); err != nil {
			t.Errorf("NewSourceList(%+v): %v", opts, err)
		}
		for _, info := range c.Infos() {
			if info.Publisher == "" || info.Licence == "" || !strings.HasPrefix(info.URL, "https://") {
				t.Errorf("info of %s = %+v, want a publisher, a licence and an HTTPS page", info.Name, info)
			}
		}
	}
}
