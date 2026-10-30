// SPDX-License-Identifier: AGPL-3.0-only

package catalog_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/domain"
)

func sources(c *catalog.Catalog) []domain.Source {
	var got []domain.Source
	for _, f := range c.Fetchers() {
		got = append(got, f.Spec().Source)
	}
	return got
}

func TestCatalogListsTheSourcesTheConfigurationTurnsOn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts catalog.Options
		want []domain.Source
		host string
		has  bool
	}{
		{"default", catalog.Options{}, []domain.Source{"AMF", "AMF PSAN", "AMF biens divers", "ARCEP"}, "www.abe-infoservice.fr", false},
		{"with ACPR", catalog.Options{ACPR: true}, []domain.Source{"AMF", "AMF PSAN", "AMF biens divers", "ACPR", "ARCEP"}, "www.abe-infoservice.fr", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := catalog.New(catalog.Offline{}, tt.opts)
			if got := sources(c); !slices.Equal(got, tt.want) {
				t.Errorf("sources = %v, want %v", got, tt.want)
			}
			if got := slices.Contains(catalog.Hosts(tt.opts), tt.host); got != tt.has {
				t.Errorf("Hosts() has %s = %t, want %t", tt.host, got, tt.has)
			}
		})
	}
}

func TestCatalogPlansEverySourceWithAPositiveInterval(t *testing.T) {
	t.Parallel()
	c := catalog.New(catalog.Offline{}, catalog.Options{ACPR: true})
	plans := c.Plans()
	if got, want := len(plans), len(c.Fetchers()); got != want {
		t.Fatalf("%d plans for %d sources", got, want)
	}
	for i, p := range plans {
		if p.Source != c.Fetchers()[i].Spec().Source || p.Every <= 0 {
			t.Errorf("plan %d = %+v, want source %s with a positive interval", i, p, c.Fetchers()[i].Spec().Source)
		}
	}
}

func TestCatalogHostsCoverEverySourceOnce(t *testing.T) {
	t.Parallel()
	hosts := catalog.Hosts(catalog.Options{ACPR: true})
	for _, h := range []string{"www.data.gouv.fr", "object-api.infra.data.gouv.fr", "www.abe-infoservice.fr", "extranet.arcep.fr"} {
		if !slices.Contains(hosts, h) {
			t.Errorf("Hosts() = %v, missing %s", hosts, h)
		}
	}
	if sorted := slices.Compact(slices.Sorted(slices.Values(hosts))); len(sorted) != len(hosts) {
		t.Errorf("Hosts() = %v, has duplicates", hosts)
	}
}

func TestOfflineNeverFetches(t *testing.T) {
	t.Parallel()
	c := catalog.New(catalog.Offline{}, catalog.Options{})
	for _, f := range c.Fetchers() {
		if _, err := f.Fetch(t.Context()); !errors.Is(err, catalog.ErrOffline) {
			t.Errorf("%s Fetch() = %v, want %v", f.Spec().Source, err, catalog.ErrOffline)
		}
	}
}
