// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func authorisation(name string, day int) domain.Authorisation {
	return domain.Authorisation{
		Name: name,
		From: time.Date(2020, time.March, day, 0, 0, 0, 0, time.UTC),
	}
}

// testWhitelist lists two sites, an address, a phone number and a SIREN.
func testWhitelist(t testing.TB) *domain.Whitelist {
	t.Helper()
	w := domain.NewWhitelist()
	w.AddSite(mustDomain(t, "www.coinhouse.com"), authorisation("COINHOUSE SAS", 17))
	w.AddSite(mustDomain(t, "fr.dfi-labs.com"), authorisation("DFI LABS", 2))
	w.AddEmail(mustEmail(t, "support@coinhouse.com"), authorisation("COINHOUSE SAS", 17))
	w.AddPhone(mustPhone(t, "+33 1 23 45 67 89"), authorisation("COINHOUSE SAS", 17))
	w.AddSIREN(mustSIREN(t, "120027016"), authorisation("COINHOUSE SAS", 17))
	w.AddSIREN(mustSIREN(t, "120027016"), authorisation("COINHOUSE CUSTODY SERVICES SAS", 18))
	return w
}

func TestWhitelistAuthorisations(t *testing.T) {
	t.Parallel()

	coinhouse := []domain.Authorisation{authorisation("COINHOUSE SAS", 17)}
	tests := []struct {
		name string
		in   domain.Input
		want []domain.Authorisation
	}{
		{"site of a www entry", mustDomain(t, "coinhouse.com"), coinhouse},
		{"subdomain of a listed site", mustDomain(t, "app.coinhouse.com"), coinhouse},
		{"parent of a listed site", mustDomain(t, "dfi-labs.com"), nil},
		{"look-alike", mustDomain(t, "coinhouse.co"), nil},
		{"listed address", mustEmail(t, "support@coinhouse.com"), coinhouse},
		{"other address of a listed site", mustEmail(t, "contact@coinhouse.com"), nil},
		{"listed phone number", mustPhone(t, "01 23 45 67 89"), coinhouse},
		{"other phone number", mustPhone(t, "01 23 45 67 88"), nil},
		{"listed SIREN, every authorisation", mustSIREN(t, "120 027 016"), []domain.Authorisation{
			authorisation("COINHOUSE SAS", 17), authorisation("COINHOUSE CUSTODY SERVICES SAS", 18),
		}},
		{"SIRET of a listed SIREN", mustSIRET(t, "12002701600563"), []domain.Authorisation{
			authorisation("COINHOUSE SAS", 17), authorisation("COINHOUSE CUSTODY SERVICES SAS", 18),
		}},
		{"other SIREN", mustSIREN(t, "732829320"), nil},
	}
	w := testWhitelist(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := w.Authorisations(tt.in); !slices.EqualFunc(got, tt.want, sameAuthorisation) {
				t.Errorf("Authorisations(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestAuthorisationEnded(t *testing.T) {
	t.Parallel()

	running := authorisation("COINHOUSE SAS", 17)
	ended := running
	ended.Until = time.Date(2026, time.May, 7, 0, 0, 0, 0, time.UTC)
	if running.Ended() {
		t.Error("an authorisation without an end date has ended")
	}
	if !ended.Ended() {
		t.Error("an authorisation with an end date runs")
	}
}

func sameAuthorisation(a, b domain.Authorisation) bool {
	return a.Name == b.Name && a.Number == b.Number && a.Kind == b.Kind && a.From.Equal(b.From) && a.Until.Equal(b.Until)
}

func BenchmarkWhitelistMiss(b *testing.B) {
	w := testWhitelist(b)
	d := mustDomain(b, "a.b.exemple-absent.fr")
	b.ReportAllocs()
	for b.Loop() {
		if w.Authorisations(d) != nil {
			b.Fatal("unexpected authorisation")
		}
	}
}

func mustSIREN(t testing.TB, s string) domain.SIREN { return must(t, s, domain.ParseSIREN) }

func mustSIRET(t testing.TB, s string) domain.SIRET { return must(t, s, domain.ParseSIRET) }
