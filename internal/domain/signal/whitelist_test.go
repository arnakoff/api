// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

const (
	whitelistsURL = "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-blanches"
	impersonation = "Des sites imitent des acteurs enregistrés : vérifie l'adresse exacte sur le site de l'AMF avant de verser de l'argent."
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func provider(name, kind string, from, until time.Time) domain.Authorisation {
	return domain.Authorisation{Name: name, Number: "E2020-001", Kind: kind, From: from, Until: until}
}

func TestWhitelistSignal(t *testing.T) {
	t.Parallel()

	psan, offers := signal.AMFPSANWhitelist(), signal.AMFOffersWhitelist()
	tests := []struct {
		name  string
		list  signal.Whitelist
		auths []domain.Authorisation
		want  domain.Signal
	}{
		{
			"running authorisation",
			psan,
			[]domain.Authorisation{provider("COINHOUSE SAS", "Agrément MICA", day(2020, time.March, 17), time.Time{})},
			domain.Signal{
				Code:    "amf_registered",
				Level:   domain.LevelInfo,
				Message: "Figure sur la liste des prestataires sur actifs numériques de l'AMF : COINHOUSE SAS (Agrément MICA) depuis le 17/03/2020.",
				Advice:  impersonation,
				Source:  "AMF PSAN",
			},
		},
		{
			"registered offer",
			offers,
			[]domain.Authorisation{{Name: "ECOTREE", Number: "D-18-01", From: day(2018, time.February, 26)}},
			domain.Signal{
				Code:    "amf_registered",
				Level:   domain.LevelInfo,
				Message: "Figure sur la liste des offres de biens divers enregistrées par l'AMF : ECOTREE (n° D-18-01) depuis le 26/02/2018.",
				Advice:  impersonation,
				Source:  "AMF biens divers",
			},
		},
		{
			"running and ended: the running ones only",
			psan,
			[]domain.Authorisation{
				provider("COINHOUSE SAS", "Enregistrement", day(2020, time.March, 17), day(2026, time.May, 7)),
				provider("COINHOUSE SAS", "Agrément MICA", day(2026, time.May, 7), time.Time{}),
			},
			domain.Signal{
				Code:    "amf_registered",
				Level:   domain.LevelInfo,
				Message: "Figure sur la liste des prestataires sur actifs numériques de l'AMF : COINHOUSE SAS (Agrément MICA) depuis le 07/05/2026.",
				Advice:  impersonation,
				Source:  "AMF PSAN",
			},
		},
		{
			"ended: first start and last end, each provider once",
			psan,
			[]domain.Authorisation{
				provider("COINHOUSE SAS", "Enregistrement", day(2020, time.March, 17), day(2026, time.May, 7)),
				provider("COINHOUSE CUSTODY SERVICES SAS", "Enregistrement", day(2021, time.June, 1), day(2025, time.July, 1)),
				provider("COINHOUSE SAS", "Enregistrement", day(2020, time.March, 17), day(2026, time.May, 7)),
			},
			domain.Signal{
				Code:    "amf_registration_ended",
				Level:   domain.LevelInfo,
				Message: "A figuré sur la liste des prestataires sur actifs numériques de l'AMF : COINHOUSE SAS (Enregistrement), COINHOUSE CUSTODY SERVICES SAS (Enregistrement) du 17/03/2020 au 07/05/2026.",
				Source:  "AMF PSAN",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := tt.list.Signal(tt.auths, observed)
			if !ok {
				t.Fatalf("Signal() gave none, want one")
			}
			want := tt.want
			want.SourceURL, want.ObservedAt = whitelistsURL, observed
			if got != want {
				t.Errorf("Signal() =\n%+v\nwant\n%+v", got, want)
			}
		})
	}
}

func TestWhitelistSignalNeedsAnAuthorisation(t *testing.T) {
	t.Parallel()

	if got, ok := signal.AMFPSANWhitelist().Signal(nil, observed); ok {
		t.Errorf("Signal(nil) = %+v, want none", got)
	}
}

func TestWhitelistSources(t *testing.T) {
	t.Parallel()

	if got := signal.AMFPSANWhitelist().Source(); got != signal.PSANSource {
		t.Errorf("PSAN source = %q, want %q", got, signal.PSANSource)
	}
	if got := signal.AMFOffersWhitelist().Source(); got != signal.OffersSource {
		t.Errorf("offers source = %q, want %q", got, signal.OffersSource)
	}
}
