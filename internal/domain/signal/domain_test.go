// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

const (
	rdapURL    = "https://rdap.nic.fr/domain/exemple.fr"
	recentText = "Un site créé il y a quelques jours n'a pas d'historique : vérifie qui l'exploite avant de verser de l'argent."
)

var checkedAt = time.Date(2026, time.October, 30, 9, 0, 0, 0, time.UTC)

func registered(at time.Time) domain.Registration {
	return domain.Registration{Registered: at, URL: rdapURL}
}

func TestDomainRecentSignal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		reg  domain.Registration
		want string // the message; "" for no signal
	}{
		{
			"12 days old", "exemple.fr", registered(checkedAt.AddDate(0, 0, -12)),
			"Le domaine exemple.fr a été créé le 18/10/2026, il y a 12 jours (registre RDAP).",
		},
		{
			"subdomain: its registrable domain", "www.boutique.exemple.fr", registered(checkedAt.AddDate(0, 0, -12)),
			"Le domaine exemple.fr a été créé le 18/10/2026, il y a 12 jours (registre RDAP).",
		},
		{
			"1 day old", "exemple.fr", registered(checkedAt.Add(-36 * time.Hour)),
			"Le domaine exemple.fr a été créé le 28/10/2026, il y a 1 jour (registre RDAP).",
		},
		{
			"hours old", "exemple.fr", registered(checkedAt.Add(-time.Hour)),
			"Le domaine exemple.fr a été créé le 30/10/2026, il y a moins d'un jour (registre RDAP).",
		},
		{
			"registered at the check", "exemple.fr", registered(checkedAt),
			"Le domaine exemple.fr a été créé le 30/10/2026, il y a moins d'un jour (registre RDAP).",
		},
		{
			"just under 30 days", "exemple.fr", registered(checkedAt.Add(-signal.RecentDomainAge + time.Second)),
			"Le domaine exemple.fr a été créé le 30/09/2026, il y a 29 jours (registre RDAP).",
		},
		{"exactly 30 days", "exemple.fr", registered(checkedAt.Add(-signal.RecentDomainAge)), ""},
		{"older", "exemple.fr", registered(checkedAt.AddDate(-3, 0, 0)), ""},
		{"no registration date published", "exemple.fr", domain.Registration{URL: rdapURL}, ""},
		{"registered after the check", "exemple.fr", registered(checkedAt.Add(time.Hour)), ""},
		{"no record address", "exemple.fr", domain.Registration{Registered: checkedAt.AddDate(0, 0, -12)}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, err := domain.ParseDomain(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := signal.DomainRecent(d, tt.reg, "RDAP", checkedAt)
			if tt.want == "" {
				if ok {
					t.Fatalf("DomainRecent() = %+v, want no signal", got)
				}
				return
			}
			want := domain.Signal{
				Code:       "domain_recent",
				Level:      domain.LevelAttention,
				Message:    tt.want,
				Advice:     recentText,
				Source:     "RDAP",
				SourceURL:  rdapURL,
				ObservedAt: checkedAt,
			}
			if !ok || got != want {
				t.Errorf("DomainRecent() = %+v, %v, want %+v", got, ok, want)
			}
		})
	}
}

func TestDomainRecentSignalDatesInTheCheckLocation(t *testing.T) {
	t.Parallel()

	plus2 := time.FixedZone("UTC+2", 2*60*60)
	d, err := domain.ParseDomain("exemple.fr")
	if err != nil {
		t.Fatal(err)
	}
	reg := registered(time.Date(2026, time.October, 17, 23, 30, 0, 0, time.UTC))
	got, ok := signal.DomainRecent(d, reg, "RDAP", checkedAt.In(plus2))
	want := "Le domaine exemple.fr a été créé le 18/10/2026, il y a 12 jours (registre RDAP)."
	if !ok || got.Message != want {
		t.Errorf("DomainRecent() message = %q, %v, want %q", got.Message, ok, want)
	}
}
