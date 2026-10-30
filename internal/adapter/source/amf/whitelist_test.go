// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/domain"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// parsedFixture returns the snapshot parse gives of an AMF fixture.
func parsedFixture(t *testing.T, name string, parse func(io.Reader) (domain.Snapshot, error)) domain.Snapshot {
	t.Helper()
	snap, err := parse(bytes.NewReader(readFixture(t, name)))
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestPSANWhitelist(t *testing.T) {
	t.Parallel()

	w, err := PSANWhitelist(parsedFixture(t, psanFixture, ParsePSAN))
	if err != nil {
		t.Fatalf("PSANWhitelist() error = %v", err)
	}
	coinhouse, _ := domain.ParseDomain("coinhouse.com")
	got := w.Authorisations(coinhouse)
	want := []domain.Authorisation{
		{Name: "COINHOUSE SAS", Number: "E2020-001", Kind: "Enregistrement", From: day(2020, time.March, 17), Until: day(2026, time.May, 7)},
		{Name: "COINHOUSE SAS", Number: "A2026-013", Kind: "Agrément MICA", From: day(2026, time.May, 7)},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Authorisations(coinhouse.com) =\n%+v\nwant\n%+v", got, want)
	}
}

func TestOffersWhitelist(t *testing.T) {
	t.Parallel()

	w, err := OffersWhitelist(parsedFixture(t, offersFixture, ParseOffers))
	if err != nil {
		t.Fatalf("OffersWhitelist() error = %v", err)
	}
	phone, _ := domain.ParsePhone("01 75 85 03 22")
	got := w.Authorisations(phone)
	want := []domain.Authorisation{{Name: "ECOTREE", Number: "D-18-01", From: day(2018, time.February, 26)}}
	if !slices.Equal(got, want) {
		t.Errorf("Authorisations(ECOTREE phone) = %+v, want %+v", got, want)
	}
}

func TestWhitelistsRejectMalformedRows(t *testing.T) {
	t.Parallel()

	key := []string{"siren", "815254545", "", "815254545"}
	psan := func(rest ...string) domain.Snapshot {
		return domain.Snapshot{Header: psanColumns, Rows: [][]string{append(slices.Clone(key), rest...)}}
	}
	offer := func(rest ...string) domain.Snapshot {
		return domain.Snapshot{Header: offersColumns, Rows: [][]string{append(slices.Clone(key), rest...)}}
	}
	tests := []struct {
		name  string
		build func(domain.Snapshot) (*domain.Whitelist, error)
		snap  domain.Snapshot
	}{
		{"PSAN start date", PSANWhitelist, psan("E1", "X", "France", "Enregistrement", "Agréé", "17/03/2020", "")},
		{"PSAN end date", PSANWhitelist, psan("E1", "X", "France", "Enregistrement", "Radié", "2020-03-17", "07/05/2026")},
		{"PSAN withdrawn without end", PSANWhitelist, psan("E1", "X", "France", "Enregistrement", "Radié", "2020-03-17", "")},
		{"PSAN unknown status", PSANWhitelist, psan("E1", "X", "France", "Enregistrement", "Suspendu", "2020-03-17", "")},
		{"offer date", OffersWhitelist, offer("D-18-01", "ECOTREE", "26/02/2018")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := tt.build(tt.snap); !errors.Is(err, listing.ErrSnapshot) {
				t.Errorf("error = %v, want ErrSnapshot", err)
			}
		})
	}
}

func TestPSANWhitelistKeepsTheRunningEndOpen(t *testing.T) {
	t.Parallel()

	// A running authorisation is running, whatever end date it announces.
	snap := domain.Snapshot{Header: psanColumns, Rows: [][]string{{
		"siren", "815254545", "", "815254545", "E1", "X", "France", "Agrément MICA", "Agréé", "2026-05-07", "2027-07-01",
	}}}
	w, err := PSANWhitelist(snap)
	if err != nil {
		t.Fatal(err)
	}
	n, _ := domain.ParseSIREN("815254545")
	if got := w.Authorisations(n); len(got) != 1 || got[0].Ended() {
		t.Errorf("Authorisations() = %+v, want one running", got)
	}
}
