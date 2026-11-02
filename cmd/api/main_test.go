// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

// emptyLists holds no snapshot: every list is unavailable.
type emptyLists struct{}

func (emptyLists) Snapshot(domain.Source) (domain.Snapshot, bool) { return domain.Snapshot{}, false }

// A phone number is covered by no registry: the check reaches no network.
func TestNewCheckConsultsTheListsAndTheRegistries(t *testing.T) {
	t.Parallel()

	check, err := newCheck(catalog.Options{}, emptyLists{})
	if err != nil {
		t.Fatal(err)
	}
	in, err := domain.ParseInput("+33 6 12 34 56 78", domain.KindPhone)
	if err != nil {
		t.Fatal(err)
	}
	var sources []domain.Source
	for _, s := range check.Check(t.Context(), in).Sources() {
		sources = append(sources, s.Source)
	}
	for _, want := range []domain.Source{"AMF", "ARCEP", "RDAP", "SIRENE", "BODACC"} {
		if !slices.Contains(sources, want) {
			t.Errorf("sources = %v, want %s among them", sources, want)
		}
	}
	if slices.Contains(sources, "ACPR") {
		t.Errorf("sources = %v, want no ACPR while the ACPR source is off", sources)
	}
}

func TestRunRequiresTrustedProxiesWithLiveKeys(t *testing.T) {
	t.Parallel()
	if err := run(t.Context(), config.Config{APIKeyEnv: "live"}); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("run() without ARNAKOFF_TRUSTED_PROXIES error = %v, want ErrInvalid", err)
	}
}
