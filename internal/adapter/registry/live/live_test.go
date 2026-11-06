// SPDX-License-Identifier: AGPL-3.0-only

package live_test

import (
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/registry/bodacc"
	"github.com/arnakoff/api/internal/adapter/registry/live"
	"github.com/arnakoff/api/internal/adapter/registry/rdap"
	"github.com/arnakoff/api/internal/adapter/registry/sirene"
	"github.com/arnakoff/api/internal/domain"
)

func TestRegistriesAreTheThreeRegistriesOnly(t *testing.T) {
	t.Parallel()

	regs, err := live.Registries(1)
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

func TestRegistriesRejectsNoInstance(t *testing.T) {
	t.Parallel()

	if _, err := live.Registries(0); err == nil {
		t.Error("Registries(0) error = nil, want an error")
	}
}

func TestInfosDescribeTheThreeRegistries(t *testing.T) {
	t.Parallel()

	infos := live.Infos()

	want := []domain.Source{rdap.Source, sirene.Source, bodacc.Source}
	if len(infos) != len(want) {
		t.Fatalf("Infos() = %+v, want %v", infos, want)
	}
	for i, info := range infos {
		if info.Name != want[i] || info.Publisher == "" || info.Licence == "" || !strings.HasPrefix(info.URL, "https://") {
			t.Errorf("Infos()[%d] = %+v, want %s with a publisher, a licence and an HTTPS page", i, info, want[i])
		}
	}
}
