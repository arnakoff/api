// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

var observed = time.Date(2026, time.October, 30, 8, 42, 26, 0, time.UTC)

// sources holds the code, source and link of the signal of each blacklist.
var sources = map[domain.Source]domain.Signal{
	"AMF":  {Code: "amf_blacklist", Source: "AMF", SourceURL: "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde"},
	"ACPR": {Code: "acpr_blacklist", Source: "ACPR", SourceURL: "https://www.abe-infoservice.fr/liste-noire"},
}

func listed(day time.Month, d int, categories ...string) domain.Listing {
	return domain.Listing{Categories: categories, ListedOn: time.Date(2026, day, d, 0, 0, 0, 0, time.UTC)}
}

func TestBlacklistSignal(t *testing.T) {
	t.Parallel()

	amf, acpr := signal.AMFBlacklist(), signal.ACPRBlacklist()
	tests := []struct {
		name     string
		list     signal.Blacklist
		listings []domain.Listing
		want     string
	}{
		{
			"one category",
			amf,
			[]domain.Listing{listed(time.October, 8, "Forex")},
			"Figure sur la liste noire de l'AMF (catégorie Forex) depuis le 08/10/2026.",
		},
		{
			"underscores of the AMF labels read as spaces",
			amf,
			[]domain.Listing{listed(time.October, 8, "Produits_dérivés_sur_crypto-actifs")},
			"Figure sur la liste noire de l'AMF (catégorie Produits dérivés sur crypto-actifs) depuis le 08/10/2026.",
		},
		{
			"several categories",
			acpr,
			[]domain.Listing{listed(time.October, 26, "Crédits livrets paiements assurances", "Usurpation Professionnel")},
			"Figure sur la liste noire de l'ACPR (catégories Crédits livrets paiements assurances, Usurpation Professionnel) depuis le 26/10/2026.",
		},
		{
			"several listings: earliest date, each category once",
			acpr,
			[]domain.Listing{listed(time.October, 11, "Forex", "Usurpation Professionnel"), listed(time.March, 12, "Forex")},
			"Figure sur la liste noire de l'ACPR (catégories Forex, Usurpation Professionnel) depuis le 12/03/2026.",
		},
		{
			"no category",
			acpr,
			[]domain.Listing{listed(time.March, 12)},
			"Figure sur la liste noire de l'ACPR depuis le 12/03/2026.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := tt.list.Signal(tt.listings, observed)
			if !ok {
				t.Fatalf("Signal() gave none, want one")
			}
			want := sources[tt.list.Source()]
			want.Level, want.Message, want.ObservedAt = domain.LevelAlert, tt.want, observed
			if got != want {
				t.Errorf("Signal() =\n%+v\nwant\n%+v", got, want)
			}
		})
	}
}

func TestBlacklistSignalNeedsAListing(t *testing.T) {
	t.Parallel()

	if got, ok := signal.AMFBlacklist().Signal(nil, observed); ok {
		t.Errorf("Signal(nil) = %+v, want none", got)
	}
}
