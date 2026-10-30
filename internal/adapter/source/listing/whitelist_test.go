// SPDX-License-Identifier: AGPL-3.0-only

package listing_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// testHeader is the header of the white list of the tests: the common
// columns and the name.
var testHeader = listing.Columns("name")

// byName decodes the name column of a test row.
func byName(row []string) (domain.Authorisation, error) {
	return domain.Authorisation{Name: row[4]}, nil
}

func whitelistSnapshot(rows ...[]string) domain.Snapshot {
	return domain.Snapshot{Source: "test", Header: testHeader, Rows: rows, FetchedAt: fetchedAt}
}

func TestWhitelistOf(t *testing.T) {
	t.Parallel()

	w, err := listing.WhitelistOf(whitelistSnapshot(
		[]string{"domain", "www.coinhouse.com", "/fr/", "https://www.coinhouse.com/fr/", "site"},
		[]string{"email", "support@coinhouse.com", "", "support@coinhouse.com", "address"},
		[]string{"phone", "+33182882951", "", "+33 1 82 88 29 51", "phone"},
		[]string{"siren", "815254545", "", "815254545", "siren"},
	), testHeader, byName)
	if err != nil {
		t.Fatalf("WhitelistOf() error = %v", err)
	}
	tests := []struct {
		in   domain.Input
		want string
	}{
		// A site entry with a path names the provider's site.
		{mustDomain(t, "coinhouse.com"), "site"},
		{mustInput(t, "support@coinhouse.com", domain.ParseEmail), "address"},
		{mustInput(t, "01 82 88 29 51", domain.ParsePhone), "phone"},
		{mustInput(t, "815254545", domain.ParseSIREN), "siren"},
	}
	for _, tt := range tests {
		got := w.Authorisations(tt.in)
		if len(got) != 1 || got[0].Name != tt.want {
			t.Errorf("Authorisations(%v) = %v, want %s", tt.in, got, tt.want)
		}
	}
}

func TestWhitelistOfRejectsAMalformedSnapshot(t *testing.T) {
	t.Parallel()

	failing := func([]string) (domain.Authorisation, error) { return domain.Authorisation{}, listing.ErrSnapshot }
	tests := []struct {
		name   string
		snap   domain.Snapshot
		decode func([]string) (domain.Authorisation, error)
	}{
		{"other header", domain.Snapshot{Header: listing.BlacklistColumns()}, byName},
		{"short row", whitelistSnapshot([]string{"siren", "815254545", "", "815254545"}), byName},
		{"unknown kind", whitelistSnapshot([]string{"iban", "FR76", "", "FR76", "x"}), byName},
		{"invalid site", whitelistSnapshot([]string{"domain", "not a host", "", "", "x"}), byName},
		{"invalid address", whitelistSnapshot([]string{"email", "nobody", "", "", "x"}), byName},
		{"invalid phone", whitelistSnapshot([]string{"phone", "12", "", "", "x"}), byName},
		{"invalid SIREN", whitelistSnapshot([]string{"siren", "815254546", "", "", "x"}), byName},
		{"undecodable row", whitelistSnapshot([]string{"siren", "815254545", "", "", "x"}), failing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := listing.WhitelistOf(tt.snap, testHeader, tt.decode); !errors.Is(err, listing.ErrSnapshot) {
				t.Errorf("WhitelistOf() error = %v, want ErrSnapshot", err)
			}
		})
	}
}

func TestWhitelistReader(t *testing.T) {
	t.Parallel()

	var s snapshots
	build := func(snap domain.Snapshot) (*domain.Whitelist, error) {
		return listing.WhitelistOf(snap, testHeader, byName)
	}
	r := listing.NewWhitelistReader(&s, "test", build)
	if _, _, err := r.Whitelist(); !errors.Is(err, app.ErrNoSnapshot) {
		t.Fatalf("Whitelist() without snapshot error = %v, want ErrNoSnapshot", err)
	}
	s.put(whitelistSnapshot([]string{"siren", "815254545", "", "815254545", "COINHOUSE SAS"}))
	w, at, err := r.Whitelist()
	if err != nil || !at.Equal(fetchedAt) {
		t.Fatalf("Whitelist() = _, %v, %v; want %v, nil", at, err, fetchedAt)
	}
	got := w.Authorisations(mustInput(t, "815254545", domain.ParseSIREN))
	if want := []string{"COINHOUSE SAS"}; !slices.Equal(names(got), want) {
		t.Errorf("Authorisations() = %v, want %v", names(got), want)
	}
}

func names(auths []domain.Authorisation) []string {
	var out []string
	for _, a := range auths {
		out = append(out, a.Name)
	}
	return out
}

func mustInput[T domain.Input](t testing.TB, s string, parse func(string) (T, error)) T {
	t.Helper()
	v, err := parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
