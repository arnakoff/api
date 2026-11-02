// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

const (
	sireneURL     = "https://recherche-entreprises.api.gouv.fr/search?include=siege&minimal=true&per_page=5&q=487568446"
	noticeURL     = "https://www.bodacc.fr/pages/annonces-commerciales-detail/?q.id=id:A202601993983"
	companyAdvice = "Vérifie que l'entreprise exerce encore avant de lui verser de l'argent."
)

func TestCompanyClosedSignal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		company domain.Company
		want    string // the message; "" for no signal
	}{
		{
			"closed on a date",
			domain.Company{Name: "PAYTOP", HeadOfficeClosed: true, HeadOfficeClosedOn: day(2024, time.March, 12), URL: sireneURL},
			"Le siège de PAYTOP est fermé depuis le 12/03/2024 (répertoire SIRENE).",
		},
		{
			"closed, no date published",
			domain.Company{Name: "PAYTOP", HeadOfficeClosed: true, URL: sireneURL},
			"Le siège de PAYTOP est fermé (répertoire SIRENE).",
		},
		{"open", domain.Company{Name: "PAYTOP", URL: sireneURL}, ""},
		{"no record address", domain.Company{Name: "PAYTOP", HeadOfficeClosed: true}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := signal.CompanyClosed(tt.company, "SIRENE", checkedAt)
			if tt.want == "" {
				if ok {
					t.Fatalf("CompanyClosed() = %+v, want no signal", got)
				}
				return
			}
			want := domain.Signal{
				Code:       "company_closed",
				Level:      domain.LevelAttention,
				Message:    tt.want,
				Advice:     companyAdvice,
				Source:     "SIRENE",
				SourceURL:  sireneURL,
				ObservedAt: checkedAt,
			}
			if !ok || got != want {
				t.Errorf("CompanyClosed() = %+v, %v, want %+v", got, ok, want)
			}
		})
	}
}

func notice(kind domain.NoticeKind, published time.Time, url string) domain.Notice {
	return domain.Notice{ID: "A202601993983", Kind: kind, Published: published, URL: url}
}

func TestCompanyProcedureSignal(t *testing.T) {
	t.Parallel()

	older := notice(domain.NoticeConciliation, day(2023, time.October, 17), "https://www.bodacc.fr/older")
	tests := []struct {
		name    string
		notices []domain.Notice
		want    string // the message; "" for no signal
	}{
		{
			"collective procedure",
			[]domain.Notice{notice(domain.NoticeCollectiveProcedure, day(2026, time.February, 5), noticeURL)},
			"Le BODACC a publié le 05/02/2026 un avis de la famille « Procédures collectives » sur cette entreprise.",
		},
		{
			"radiation",
			[]domain.Notice{notice(domain.NoticeRadiation, day(2026, time.February, 5), noticeURL)},
			"Le BODACC a publié le 05/02/2026 un avis de la famille « Radiations » sur cette entreprise.",
		},
		{
			"conciliation",
			[]domain.Notice{notice(domain.NoticeConciliation, day(2026, time.February, 5), noticeURL)},
			"Le BODACC a publié le 05/02/2026 un avis de la famille « Procédures de conciliation » sur cette entreprise.",
		},
		{
			"professional recovery",
			[]domain.Notice{notice(domain.NoticeProfessionalRecovery, day(2026, time.February, 5), noticeURL)},
			"Le BODACC a publié le 05/02/2026 un avis de la famille « Procédures de rétablissement professionnel » sur cette entreprise.",
		},
		{
			"the most recent notice, whatever the order",
			[]domain.Notice{older, notice(domain.NoticeRadiation, day(2026, time.February, 5), noticeURL)},
			"Le BODACC a publié le 05/02/2026 un avis de la famille « Radiations » sur cette entreprise.",
		},
		{"no notice", nil, ""},
		{"unknown family", []domain.Notice{notice(0, day(2026, time.February, 5), noticeURL)}, ""},
		{"no notice address", []domain.Notice{notice(domain.NoticeRadiation, day(2026, time.February, 5), "")}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := signal.CompanyProcedure(tt.notices, "BODACC", checkedAt)
			if tt.want == "" {
				if ok {
					t.Fatalf("CompanyProcedure() = %+v, want no signal", got)
				}
				return
			}
			want := domain.Signal{
				Code:       "company_procedure",
				Level:      domain.LevelAttention,
				Message:    tt.want,
				Advice:     companyAdvice,
				Source:     "BODACC",
				SourceURL:  noticeURL,
				ObservedAt: checkedAt,
			}
			if !ok || got != want {
				t.Errorf("CompanyProcedure() = %+v, %v, want %+v", got, ok, want)
			}
		})
	}
}
