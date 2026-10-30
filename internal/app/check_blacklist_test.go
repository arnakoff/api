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

var fetchedAt = time.Date(2026, time.October, 30, 8, 42, 26, 0, time.UTC)

// fakeBlacklist serves list, fetched at fetchedAt, or err.
type fakeBlacklist struct {
	list *domain.Blacklist
	err  error
}

func (f fakeBlacklist) Blacklist() (*domain.Blacklist, time.Time, error) {
	if f.err != nil {
		return nil, time.Time{}, f.err
	}
	return f.list, fetchedAt, nil
}

// blacklistOf lists the sites and addresses entries, each in category on
// 08/10/2026.
func blacklistOf(t testing.TB, category string, entries ...string) *domain.Blacklist {
	t.Helper()
	b := domain.NewBlacklist()
	l := domain.Listing{Categories: []string{category}, ListedOn: time.Date(2026, time.October, 8, 0, 0, 0, 0, time.UTC)}
	for _, e := range entries {
		switch v := parse(t, e).(type) {
		case domain.Domain:
			b.AddSite(v, l)
		case domain.Email:
			b.AddEmail(v, l)
		default:
			t.Fatalf("%q is neither a site nor an address", e)
		}
	}
	return b
}

// blacklists returns an AMF and an ACPR blacklist served by amf and acpr.
func blacklists(amf, acpr app.BlacklistReader) []app.Blacklist {
	return []app.Blacklist{
		{Rule: signal.AMFBlacklist(), Reader: amf},
		{Rule: signal.ACPRBlacklist(), Reader: acpr},
	}
}

func TestCheckReadsTheBlacklists(t *testing.T) {
	t.Parallel()

	amf := fakeBlacklist{list: blacklistOf(t, "Forex", "armen.ltd", "contact@paris-majeste.com")}
	acpr := fakeBlacklist{list: blacklistOf(t, "Usurpation Professionnel", "armen.ltd")}
	check := app.NewCheck(app.Registries{Blacklists: blacklists(amf, acpr)}, &clock{now: t0})
	tests := []struct {
		in      string
		codes   []string
		amf, ac domain.SourceStatus
	}{
		{"https://www.armen.ltd/contact", []string{"amf_blacklist", "acpr_blacklist"}, okState, okState},
		{"contact@paris-majeste.com", []string{"amf_blacklist"}, okState, okState},
		{"exemple.fr", nil, okState, okState},
		{"+33612345678", nil, naState, naState},
		{"120027016", nil, naState, naState},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			got := check.Check(context.Background(), parse(t, tt.in))

			if codes := signalCodes(got.Signals()); !slices.Equal(codes, tt.codes) {
				t.Errorf("signal codes = %v, want %v", codes, tt.codes)
			}
			want := []domain.SourceState{{Source: "AMF", Status: tt.amf}, {Source: "ACPR", Status: tt.ac}}
			if !slices.Equal(got.Sources(), want) {
				t.Errorf("Sources() = %v, want %v", got.Sources(), want)
			}
		})
	}
}

func TestCheckBlacklistSignal(t *testing.T) {
	t.Parallel()

	amf := fakeBlacklist{list: blacklistOf(t, "Forex", "armen.ltd")}
	check := app.NewCheck(app.Registries{Blacklists: blacklists(amf, fakeBlacklist{list: domain.NewBlacklist()})}, &clock{now: t0})

	got := check.Check(context.Background(), parse(t, "armen.ltd")).Signals()

	want := []domain.Signal{{
		Code:       "amf_blacklist",
		Level:      domain.LevelAlert,
		Message:    "Figure sur la liste noire de l'AMF (catégorie Forex) depuis le 08/10/2026.",
		Source:     "AMF",
		SourceURL:  "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde",
		ObservedAt: fetchedAt,
	}}
	if !slices.Equal(got, want) {
		t.Errorf("Signals() = %+v, want %+v", got, want)
	}
}

func TestCheckReportsABlacklistWithoutSnapshotUnavailable(t *testing.T) {
	t.Parallel()

	amf := fakeBlacklist{err: fmt.Errorf("amf: %w", app.ErrNoSnapshot)}
	acpr := fakeBlacklist{list: blacklistOf(t, "Forex", "armen.ltd")}
	check := app.NewCheck(app.Registries{Blacklists: blacklists(amf, acpr)}, &clock{now: t0})

	got := check.Check(context.Background(), parse(t, "armen.ltd"))

	if codes := signalCodes(got.Signals()); !slices.Equal(codes, []string{"acpr_blacklist"}) {
		t.Errorf("signal codes = %v, want the ACPR one alone", codes)
	}
	want := []domain.SourceState{{Source: "AMF", Status: domain.SourceUnavailable}, {Source: "ACPR", Status: okState}}
	if !slices.Equal(got.Sources(), want) {
		t.Errorf("Sources() = %v, want %v", got.Sources(), want)
	}
}

func TestCheckListsTheBlacklistsBeforeTheRegistries(t *testing.T) {
	t.Parallel()

	regs := newRegistries().all()
	regs.Blacklists = blacklists(fakeBlacklist{list: domain.NewBlacklist()}, fakeBlacklist{list: domain.NewBlacklist()})

	got := app.NewCheck(regs, &clock{now: t0}).Check(context.Background(), parse(t, "exemple.fr")).Sources()

	want := append([]domain.SourceState{{Source: "AMF", Status: okState}, {Source: "ACPR", Status: okState}}, states(okState, naState, naState)...)
	if !slices.Equal(got, want) {
		t.Errorf("Sources() = %v, want %v", got, want)
	}
}

func signalCodes(signals []domain.Signal) []string {
	var out []string
	for _, s := range signals {
		out = append(out, s.Code)
	}
	return out
}
