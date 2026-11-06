// SPDX-License-Identifier: AGPL-3.0-only

package assembly_test

import (
	"context"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/reference/official"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// emptyStore is a store in a fresh directory: no snapshot is in service.
func emptyStore(t *testing.T) *store.Disk {
	t.Helper()
	disk, err := store.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return disk
}

// newService returns the service of the lists opts turns on, with no
// snapshot in service, and of the live registries.
func newService(t *testing.T, opts catalog.Options) assembly.Service {
	t.Helper()
	svc, err := assembly.Live(opts, emptyStore(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// checked returns the sources a check of a phone number reports. A phone
// number is covered by no registry: the check reaches no network.
func checked(t *testing.T, check *app.Check) []domain.Source {
	t.Helper()
	in, err := domain.ParseInput("+33 6 12 34 56 78", domain.KindPhone)
	if err != nil {
		t.Fatal(err)
	}
	var sources []domain.Source
	for _, s := range check.Check(t.Context(), in).Sources() {
		sources = append(sources, s.Source)
	}
	return sources
}

// listed returns the sources a source list shows.
func listed(l *app.SourceList) []domain.Source {
	var sources []domain.Source
	for _, s := range l.Sources() {
		sources = append(sources, s.Info.Name)
	}
	return sources
}

func TestNewConsultsTheListsOptsTurnsOnAndTheRegistries(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		opts catalog.Options
		acpr bool
	}{
		{"default", catalog.Options{}, false},
		{"acpr", catalog.Options{ACPR: true}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(t, tt.opts)
			wantSources(t, "checked", checked(t, svc.Check), tt.acpr)
			wantSources(t, "listed", listed(svc.Sources), tt.acpr)
		})
	}
}

// wantSources reports an error unless sources hold the default lists and
// the registries, and the ACPR blacklist exactly when acpr is set.
func wantSources(t *testing.T, name string, sources []domain.Source, acpr bool) {
	t.Helper()
	for _, want := range []domain.Source{"AMF", "ARCEP", "RDAP", "SIRENE", "BODACC"} {
		if !slices.Contains(sources, want) {
			t.Errorf("%s sources = %v, want %s among them", name, sources, want)
		}
	}
	if got := slices.Contains(sources, "ACPR"); got != acpr {
		t.Errorf("%s sources = %v, ACPR among them = %t, want %t", name, sources, got, acpr)
	}
}

func TestNewListsTheIBANCountry(t *testing.T) {
	t.Parallel()
	if sources := listed(newService(t, catalog.Options{}).Sources); !slices.Contains(sources, "IBAN") {
		t.Errorf("listed sources = %v, want IBAN among them", sources)
	}
}

func TestNewSyncsTheSourcesItLists(t *testing.T) {
	t.Parallel()
	svc := newService(t, catalog.Options{ACPR: true})
	if _, ok := svc.Sync.Snapshot("ACPR"); ok {
		t.Error("Snapshot(ACPR) in service in an empty store, want none")
	}
	for _, s := range svc.Sources.Sources() {
		if s.Info.Name == "ACPR" && !s.Synced {
			t.Errorf("ACPR listed as not synced, want synced")
		}
	}
}

// domains is a fake domain registry.
type domains struct{}

func (domains) Source() domain.Source { return "fake" }

func (domains) Lookup(context.Context, domain.Domain) (domain.Registration, error) {
	return domain.Registration{}, nil
}

func TestListsKeepsTheRegistriesOfRegs(t *testing.T) {
	t.Parallel()
	s, err := assembly.Sync(catalog.Options{}, emptyStore(t))
	if err != nil {
		t.Fatal(err)
	}
	regs, err := assembly.Lists(catalog.Options{}, s, app.Registries{Domains: domains{}})
	if err != nil {
		t.Fatal(err)
	}
	if regs.Domains != (domains{}) {
		t.Errorf("Domains = %v, want the registry of regs", regs.Domains)
	}
	if len(regs.Blacklists) == 0 || len(regs.Whitelists) == 0 || len(regs.Numbering) == 0 {
		t.Errorf("Lists() = %+v, want the blacklists, the white lists and the numbering plan", regs)
	}
}

func TestNewReadsTheOfficialDomains(t *testing.T) {
	t.Parallel()
	svc, err := assembly.New(catalog.Options{}, emptyStore(t), app.Registries{})
	if err != nil {
		t.Fatal(err)
	}
	in, err := domain.ParseInput("ameli-remboursement.com", domain.KindDomain)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(svc.Check.Check(t.Context(), in).Signals(), "domain_lookalike") {
		t.Error("check of ameli-remboursement.com gives no domain_lookalike, want one")
	}
	if sources := listed(svc.Sources); !slices.Contains(sources, official.Source) {
		t.Errorf("listed sources = %v, want %s among them", sources, official.Source)
	}
}

// hasCode tells whether signals hold one of code.
func hasCode(signals []domain.Signal, code string) bool {
	return slices.ContainsFunc(signals, func(s domain.Signal) bool { return string(s.Code) == code })
}
