// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

const (
	sireneURL   = "https://recherche-entreprises.api.gouv.fr/search?include=siege&minimal=true&per_page=5&q=120027016"
	noticeURL   = "https://www.bodacc.fr/pages/annonces-commerciales-detail/?q.id=id:A202601993983"
	companyText = "Vérifie que l'entreprise exerce encore avant de lui verser de l'argent."
)

var (
	// closedCompany is the SIRENE record of a company whose head office
	// closed.
	closedCompany = domain.Company{
		Name:               "PAYTOP",
		HeadOfficeClosed:   true,
		HeadOfficeClosedOn: time.Date(2024, time.March, 12, 0, 0, 0, 0, time.UTC),
		URL:                sireneURL,
	}
	// procedure is a BODACC collective procedure notice.
	procedure = domain.Notice{
		ID:        "A202601993983",
		Kind:      domain.NoticeCollectiveProcedure,
		Published: time.Date(2026, time.February, 5, 0, 0, 0, 0, time.UTC),
		URL:       noticeURL,
	}
)

func TestCheckFlagsAClosedCompanyUnderProcedure(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"120027016", "12002701600563"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()

			r := newRegistries()
			r.companies.company = closedCompany
			r.notices.notices = []domain.Notice{procedure}
			got := app.NewCheck(r.all(), &clock{now: t0}).Check(context.Background(), parse(t, in)).Signals()

			want := []domain.Signal{{
				Code:       "company_closed",
				Level:      domain.LevelAttention,
				Message:    "Le siège de PAYTOP est fermé depuis le 12/03/2024 (répertoire SIRENE).",
				Advice:     companyText,
				Source:     "SIRENE",
				SourceURL:  sireneURL,
				ObservedAt: t0,
			}, {
				Code:       "company_procedure",
				Level:      domain.LevelAttention,
				Message:    "Le BODACC a publié le 05/02/2026 un avis de la famille « Procédures collectives » sur cette entreprise.",
				Advice:     companyText,
				Source:     "BODACC",
				SourceURL:  noticeURL,
				ObservedAt: t0,
			}}
			if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Errorf("Signals() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestCheckGivesNoCompanySignalWithoutAFact(t *testing.T) {
	t.Parallel()

	open := closedCompany
	open.HeadOfficeClosed, open.HeadOfficeClosedOn = false, time.Time{}
	down := errors.New("connection reset")
	tests := []struct {
		name       string
		company    domain.Company
		notices    []domain.Notice
		companyErr error
		noticeErr  error
	}{
		{"open, no notice", open, nil, nil, nil},
		{"registries down", closedCompany, []domain.Notice{procedure}, down, down},
		{"not found", domain.Company{}, nil, app.ErrNotFound, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newRegistries()
			r.companies.company, r.companies.err = tt.company, tt.companyErr
			r.notices.notices, r.notices.err = tt.notices, tt.noticeErr
			got := app.NewCheck(r.all(), &clock{now: t0}).Check(context.Background(), parse(t, "120027016")).Signals()

			if len(got) != 0 {
				t.Errorf("Signals() = %+v, want none", got)
			}
		})
	}
}
