// SPDX-License-Identifier: AGPL-3.0-only

package catalog_test

import (
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/domain"
)

func TestListsHoldTheListsAndNoRegistry(t *testing.T) {
	t.Parallel()

	opts := catalog.Options{ACPR: true}
	regs := catalog.Lists(opts, noSnapshot{}, domain.Platforms{})
	if got, want := len(regs.Blacklists), len(catalog.Blacklists(opts, noSnapshot{})); got != want {
		t.Errorf("%d blacklists, want %d", got, want)
	}
	if got, want := len(regs.Whitelists), len(catalog.Whitelists(noSnapshot{}, domain.Platforms{})); got != want {
		t.Errorf("%d white lists, want %d", got, want)
	}
	if len(regs.Numbering) != 1 {
		t.Errorf("%d numbering plans, want 1", len(regs.Numbering))
	}
	if regs.Domains != nil || regs.Companies != nil || regs.Notices != nil {
		t.Errorf("Lists holds registries: %+v", regs)
	}
}
