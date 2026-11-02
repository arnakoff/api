// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

var (
	web         = []domain.Kind{domain.KindDomain, domain.KindEmail}
	company     = []domain.Kind{domain.KindSIREN, domain.KindSIRET}
	whitelisted = []domain.Kind{domain.KindDomain, domain.KindEmail, domain.KindPhone, domain.KindSIREN, domain.KindSIRET}
)

func TestCatalogListsEverySignalOncePerSourceInDocumentOrder(t *testing.T) {
	t.Parallel()

	want := []struct {
		code   string
		level  domain.Level
		inputs []domain.Kind
		source domain.Source
	}{
		{"amf_blacklist", domain.LevelAlert, web, "AMF"},
		{"acpr_blacklist", domain.LevelAlert, web, "ACPR"},
		{"amf_registered", domain.LevelInfo, whitelisted, "AMF PSAN"},
		{"amf_registered", domain.LevelInfo, whitelisted, "AMF biens divers"},
		{"amf_registration_ended", domain.LevelInfo, whitelisted, "AMF PSAN"},
		{"domain_recent", domain.LevelAttention, web, "RDAP"},
		{"domain_lookalike", domain.LevelAttention, web, "Domaines officiels"},
		{"phone_operator", domain.LevelInfo, []domain.Kind{domain.KindPhone}, "ARCEP"},
		{"phone_unassigned", domain.LevelAttention, []domain.Kind{domain.KindPhone}, "ARCEP"},
		{"company_closed", domain.LevelAttention, company, "SIRENE"},
		{"company_procedure", domain.LevelAttention, company, "BODACC"},
		{"iban_country", domain.LevelInfo, []domain.Kind{domain.KindIBAN}, "IBAN"},
	}

	got := signal.Catalog()

	if len(got) != len(want) {
		t.Fatalf("Catalog() has %d definitions, want %d", len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		if g.Code != w.code || g.Level != w.level || g.Source != w.source || !slices.Equal(g.Inputs, w.inputs) {
			t.Errorf("Catalog()[%d] = %s %v %v %s, want %s %v %v %s",
				i, g.Code, g.Level, g.Inputs, g.Source, w.code, w.level, w.inputs, w.source)
		}
		if g.Condition == "" {
			t.Errorf("Catalog()[%d] %s has no condition", i, g.Code)
		}
	}
}

func TestCatalogReturnsACopy(t *testing.T) {
	t.Parallel()

	first := signal.Catalog()
	first[0].Code = "changed"
	first[0].Inputs[0] = domain.KindIBAN

	again := signal.Catalog()
	if again[0].Code != "amf_blacklist" || again[0].Inputs[0] != domain.KindDomain {
		t.Errorf("Catalog() = %+v after a change to an earlier result, want it unchanged", again[0])
	}
}

func TestDefinitionInputNamesAreTheContractInputTypes(t *testing.T) {
	t.Parallel()

	d := signal.Definition{Inputs: []domain.Kind{domain.KindDomain, domain.KindEmail, domain.KindIBAN}}
	if got, want := d.InputNames(), []string{"domain", "email", "iban"}; !slices.Equal(got, want) {
		t.Errorf("InputNames() = %v, want %v", got, want)
	}
}
