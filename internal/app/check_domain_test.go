// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// recentRDAP is the RDAP record of a domain registered 12 days before t0.
var recentRDAP = domain.Registration{Registered: t0.AddDate(0, 0, -12), URL: "https://rdap.nic.fr/domain/exemple.fr"}

func TestCheckFlagsARecentDomain(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"www.exemple.fr", "contact@exemple.fr"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()

			r := newRegistries()
			r.domains.reg = recentRDAP
			got := app.NewCheck(r.all(), &clock{now: t0}).Check(context.Background(), parse(t, in)).Signals()

			want := domain.Signal{
				Code:       "domain_recent",
				Level:      domain.LevelAttention,
				Message:    "Le domaine exemple.fr a été créé le 17/10/2026, il y a 12 jours (registre RDAP).",
				Advice:     "Un site créé il y a quelques jours n'a pas d'historique : vérifie qui l'exploite avant de verser de l'argent.",
				Source:     "RDAP",
				SourceURL:  "https://rdap.nic.fr/domain/exemple.fr",
				ObservedAt: t0,
			}
			if len(got) != 1 || got[0] != want {
				t.Errorf("Signals() = %+v, want [%+v]", got, want)
			}
		})
	}
}

func TestCheckGivesNoDomainSignalWithoutARecentRecord(t *testing.T) {
	t.Parallel()

	old := recentRDAP
	old.Registered = t0.AddDate(-5, 0, 0)
	tests := []struct {
		name string
		reg  domain.Registration
		err  error
	}{
		{"old domain", old, nil},
		{"registry down", recentRDAP, errors.New("connection reset")},
		{"not found", domain.Registration{}, app.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newRegistries()
			r.domains.reg, r.domains.err = tt.reg, tt.err
			got := app.NewCheck(r.all(), &clock{now: t0}).Check(context.Background(), parse(t, "exemple.fr")).Signals()

			if len(got) != 0 {
				t.Errorf("Signals() = %+v, want none", got)
			}
		})
	}
}
