// SPDX-License-Identifier: AGPL-3.0-only

package live_test

import (
	"testing"

	"github.com/arnakoff/api/internal/adapter/registry/bodacc"
	"github.com/arnakoff/api/internal/adapter/registry/live"
	"github.com/arnakoff/api/internal/adapter/registry/rdap"
	"github.com/arnakoff/api/internal/adapter/registry/sirene"
)

func TestRegistriesAreTheThreeRegistriesOnly(t *testing.T) {
	t.Parallel()

	regs, err := live.Registries()
	if err != nil {
		t.Fatal(err)
	}
	if regs.Domains == nil || regs.Domains.Source() != rdap.Source {
		t.Errorf("Domains = %v, want the RDAP client", regs.Domains)
	}
	if regs.Companies == nil || regs.Companies.Source() != sirene.Source {
		t.Errorf("Companies = %v, want the SIRENE client", regs.Companies)
	}
	if regs.Notices == nil || regs.Notices.Source() != bodacc.Source {
		t.Errorf("Notices = %v, want the BODACC client", regs.Notices)
	}
	if len(regs.Blacklists)+len(regs.Whitelists)+len(regs.Numbering) != 0 {
		t.Errorf("Registries holds lists: %+v", regs)
	}
}
