// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

var fixtures = os.DirFS("../../../../testdata/sirene")

func readFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	b, err := fs.ReadFile(fixtures, name)
	if err != nil {
		tb.Fatalf("read fixture: %v", err)
	}
	return b
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestParseCompany(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fixture string
		siren   string
		want    domain.Company
	}{
		{
			name:    "active company",
			fixture: "insee-active.json",
			siren:   "120027016",
			want: domain.Company{
				Name:    "INSTITUT NATIONAL DE LA STATISTIQUE ET DES ETUDES ECONOMIQUES (INSEE)",
				Created: day(1983, time.March, 1),
			},
		},
		{
			name:    "closed head office",
			fixture: "closed.json",
			siren:   "814812236",
			want: domain.Company{
				Name:               "TRANSPORT & VOUS (TRANSPORT & VOUS)",
				Created:            day(2015, time.November, 9),
				HeadOfficeClosed:   true,
				HeadOfficeClosedOn: day(2018, time.June, 30),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseCompany(readFixture(t, tt.fixture), tt.siren)
			if err != nil {
				t.Fatalf("parseCompany() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("parseCompany() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseCompanyNotFound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fixture string
		siren   string
	}{
		{"no result", "no-result.json", "000000000"},
		{"only another company matches the text search", "insee-active.json", "552032534"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseCompany(readFixture(t, tt.fixture), tt.siren); !errors.Is(err, ErrNotFound) {
				t.Errorf("parseCompany() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestParseCompanyRejectsInvalidResponses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{"not JSON", `<html>`},
		{"no results field", `{}`},
		{"no name", `{"results":[{"siren":"120027016","nom_complet":"","siege":{"etat_administratif":"A"}}]}`},
		{"unknown head office state", `{"results":[{"siren":"120027016","nom_complet":"X","siege":{"etat_administratif":"C"}}]}`},
		{"bad creation date", `{"results":[{"siren":"120027016","nom_complet":"X","date_creation":"01/03/1983","siege":{"etat_administratif":"A"}}]}`},
		{"bad closing date", `{"results":[{"siren":"120027016","nom_complet":"X","siege":{"etat_administratif":"F","date_fermeture":"2018"}}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseCompany([]byte(tt.body), "120027016"); !errors.Is(err, ErrResponse) {
				t.Errorf("parseCompany(%s) error = %v, want ErrResponse", tt.body, err)
			}
		})
	}
}

func TestParseCompanyWithoutDates(t *testing.T) {
	t.Parallel()
	body := `{"results":[{"siren":"120027016","nom_complet":"X","date_creation":null,"siege":{"etat_administratif":"F","date_fermeture":null}}]}`
	got, err := parseCompany([]byte(body), "120027016")
	if err != nil {
		t.Fatalf("parseCompany() error = %v", err)
	}
	want := domain.Company{Name: "X", HeadOfficeClosed: true}
	if got != want {
		t.Errorf("parseCompany() = %+v, want %+v", got, want)
	}
}
