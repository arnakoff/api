// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// fakeWhitelist serves list, fetched at fetchedAt, or err.
type fakeWhitelist struct {
	list *domain.Whitelist
	err  error
}

func (f fakeWhitelist) Whitelist() (*domain.Whitelist, time.Time, error) {
	if f.err != nil {
		return nil, time.Time{}, f.err
	}
	return f.list, fetchedAt, nil
}

// whitelistOf lists the keys, each under a running authorisation of name
// since 17/03/2020.
func whitelistOf(t testing.TB, name string, keys ...string) *domain.Whitelist {
	t.Helper()
	w := domain.NewWhitelist()
	a := domain.Authorisation{Name: name, Number: "E2020-001", Kind: "Agrément MICA", From: time.Date(2020, time.March, 17, 0, 0, 0, 0, time.UTC)}
	for _, k := range keys {
		switch v := parse(t, k).(type) {
		case domain.Domain:
			w.AddSite(v, a)
		case domain.Email:
			w.AddEmail(v, a)
		case domain.Phone:
			w.AddPhone(v, a)
		case domain.SIREN:
			w.AddSIREN(v, a)
		default:
			t.Fatalf("%q is not a white list key", k)
		}
	}
	return w
}

// whitelists returns the PSAN and biens divers white lists served by psan
// and offers.
func whitelists(psan, offers app.WhitelistReader) []app.Whitelist {
	return []app.Whitelist{
		{Rule: signal.AMFPSANWhitelist(), Reader: psan},
		{Rule: signal.AMFOffersWhitelist(), Reader: offers},
	}
}

func TestCheckReadsTheWhitelists(t *testing.T) {
	t.Parallel()

	psan := fakeWhitelist{list: whitelistOf(t, "COINHOUSE SAS", "coinhouse.com", "support@coinhouse.com", "+33182882951", "815254545")}
	offers := fakeWhitelist{list: whitelistOf(t, "ECOTREE", "+33175850322")}
	check := app.NewCheck(app.Registries{Whitelists: whitelists(psan, offers)})
	tests := []struct {
		in    string
		codes []string
		state domain.SourceStatus
	}{
		{"https://app.coinhouse.com/login", []string{"amf_registered"}, okState},
		{"support@coinhouse.com", []string{"amf_registered"}, okState},
		{"+33182882951", []string{"amf_registered"}, okState},
		{"+33175850322", []string{"amf_registered"}, okState},
		{"815254545", []string{"amf_registered"}, okState},
		{"81525454500024", []string{"amf_registered"}, okState},
		{"exemple.fr", nil, okState},
		{"FR7630006000011234567890189", nil, naState},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			got := check.Check(context.Background(), parse(t, tt.in))

			if codes := signalCodes(got.Signals()); !slices.Equal(codes, tt.codes) {
				t.Errorf("signal codes = %v, want %v", codes, tt.codes)
			}
			want := []domain.SourceState{{Source: "AMF PSAN", Status: tt.state}, {Source: "AMF biens divers", Status: tt.state}}
			if !slices.Equal(got.Sources(), want) {
				t.Errorf("Sources() = %v, want %v", got.Sources(), want)
			}
		})
	}
}

func TestCheckReportsAWhitelistWithoutSnapshotUnavailable(t *testing.T) {
	t.Parallel()

	psan := fakeWhitelist{err: fmt.Errorf("psan: %w", app.ErrNoSnapshot)}
	check := app.NewCheck(app.Registries{Whitelists: whitelists(psan, fakeWhitelist{list: domain.NewWhitelist()})})

	got := check.Check(context.Background(), parse(t, "coinhouse.com"))

	if len(got.Signals()) != 0 {
		t.Errorf("Signals() = %v, want none", got.Signals())
	}
	want := []domain.SourceState{{Source: "AMF PSAN", Status: domain.SourceUnavailable}, {Source: "AMF biens divers", Status: okState}}
	if !slices.Equal(got.Sources(), want) {
		t.Errorf("Sources() = %v, want %v", got.Sources(), want)
	}
}

func TestCheckListsTheWhitelistsAfterTheBlacklists(t *testing.T) {
	t.Parallel()

	regs := newRegistries().all()
	regs.Blacklists = []app.Blacklist{{Rule: signal.AMFBlacklist(), Reader: fakeBlacklist{list: domain.NewBlacklist()}}}
	regs.Whitelists = whitelists(fakeWhitelist{list: domain.NewWhitelist()}, fakeWhitelist{list: domain.NewWhitelist()})

	got := app.NewCheck(regs).Check(context.Background(), parse(t, "exemple.fr")).Sources()

	want := append([]domain.SourceState{
		{Source: "AMF", Status: okState}, {Source: "AMF PSAN", Status: okState}, {Source: "AMF biens divers", Status: okState},
	}, states(okState, naState, naState)...)
	if !slices.Equal(got, want) {
		t.Errorf("Sources() = %v, want %v", got, want)
	}
}

func TestRegistriesWithoutAWhitelist(t *testing.T) {
	t.Parallel()

	regs := app.Registries{Whitelists: whitelists(fakeWhitelist{}, fakeWhitelist{})}

	got := regs.Without(func(s domain.Source) bool { return s == signal.PSANSource })

	if len(got.Whitelists) != 1 || got.Whitelists[0].Rule.Source() != signal.OffersSource {
		t.Errorf("Without(PSAN) whitelists = %v, want biens divers alone", got.Whitelists)
	}
	if len(regs.Whitelists) != 2 {
		t.Error("Without changed the registries it was called on")
	}
}
