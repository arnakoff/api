// SPDX-License-Identifier: AGPL-3.0-only

package replay_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/registry/replay"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

var (
	registered = domain.Registration{
		Registered:  time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC),
		LastChanged: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
	}
	company = domain.Company{
		Name:               "DIRECTION INTERMINISTERIELLE DU NUMERIQUE",
		Created:            time.Date(2019, 10, 25, 0, 0, 0, 0, time.UTC),
		HeadOfficeClosed:   true,
		HeadOfficeClosedOn: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
	}
	notices = []domain.Notice{{
		ID:        "A202601993983",
		Kind:      domain.NoticeCollectiveProcedure,
		Published: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC),
	}}
)

func TestRecordedAnswersReplayAfterSaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tape, err := replay.Load(dir)
	if err != nil {
		t.Fatalf("Load of an empty directory: %v", err)
	}
	live := liveRegistries()
	rec := tape.Registries(live)
	ctx := context.Background()
	mustLookup(t, rec, "www.exemple.fr", "exemple.com", "exemple.io", "120027016", "552100554")
	if err := tape.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := replay.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	play := loaded.Registries(app.Registries{})

	// The registrable domain is the key: a subdomain replays its answer.
	if got, err := play.Domains.Lookup(ctx, mustDomain(t, "exemple.fr")); err != nil || got != registered {
		t.Errorf("Lookup(exemple.fr) = %+v, %v; want %+v, nil", got, err, registered)
	}
	if _, err := play.Domains.Lookup(ctx, mustDomain(t, "exemple.com")); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("Lookup(exemple.com) error = %v, want app.ErrNotFound", err)
	}
	if _, err := play.Domains.Lookup(ctx, mustDomain(t, "exemple.io")); !errors.Is(err, app.ErrNotCovered) {
		t.Errorf("Lookup(exemple.io) error = %v, want app.ErrNotCovered", err)
	}
	if got, err := play.Companies.Lookup(ctx, mustSIREN(t, "120027016")); err != nil || got != company {
		t.Errorf("Lookup(120027016) = %+v, %v; want %+v, nil", got, err, company)
	}
	if got, err := play.Notices.Notices(ctx, mustSIREN(t, "120027016")); err != nil || !reflect.DeepEqual(got, notices) {
		t.Errorf("Notices(120027016) = %+v, %v; want %+v, nil", got, err, notices)
	}
	if _, err := play.Companies.Lookup(ctx, mustSIREN(t, "552100554")); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("Lookup(552100554) error = %v, want app.ErrNotFound", err)
	}
	if got, err := play.Notices.Notices(ctx, mustSIREN(t, "552100554")); err != nil || len(got) != 0 {
		t.Errorf("Notices(552100554) = %+v, %v; want none", got, err)
	}
}

func TestAnUnrecordedQueryIsAFailure(t *testing.T) {
	t.Parallel()

	tape, err := replay.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	play := tape.Registries(app.Registries{})
	ctx := context.Background()

	_, errD := play.Domains.Lookup(ctx, mustDomain(t, "exemple.fr"))
	_, errC := play.Companies.Lookup(ctx, mustSIREN(t, "120027016"))
	_, errN := play.Notices.Notices(ctx, mustSIREN(t, "120027016"))
	for _, err := range []error{errD, errC, errN} {
		if !errors.Is(err, replay.ErrNotRecorded) {
			t.Errorf("error = %v, want ErrNotRecorded", err)
		}
	}
}

func TestAFailureIsNotRecorded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tape, err := replay.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("connection reset")
	live := liveRegistries()
	live.Domains.(*liveDomains).err = failure

	_, err = tape.Registries(live).Domains.Lookup(context.Background(), mustDomain(t, "exemple.fr"))
	if !errors.Is(err, failure) {
		t.Errorf("recording Lookup error = %v, want the live error", err)
	}
	_, err = tape.Registries(app.Registries{}).Domains.Lookup(context.Background(), mustDomain(t, "exemple.fr"))
	if !errors.Is(err, replay.ErrNotRecorded) {
		t.Errorf("replayed error = %v, want ErrNotRecorded", err)
	}
}

func TestReplayNamesTheRegistries(t *testing.T) {
	t.Parallel()

	tape, err := replay.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := tape.Registries(app.Registries{})
	got := []domain.Source{r.Domains.Source(), r.Companies.Source(), r.Notices.Source()}
	want := []domain.Source{"RDAP", "SIRENE", "BODACC"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sources = %v, want %v", got, want)
	}
}

func TestLoadRejectsACorruptFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rdap.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := replay.Load(dir); err == nil {
		t.Error("Load of a corrupt file: want an error")
	}
}

func TestAnEmptyEntryIsAFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rdap.json"), []byte(`{"exemple.fr": {}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tape, err := replay.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tape.Registries(app.Registries{}).Domains.Lookup(context.Background(), mustDomain(t, "exemple.fr"))
	if err == nil || errors.Is(err, app.ErrNotFound) || errors.Is(err, app.ErrNotCovered) {
		t.Errorf("Lookup error = %v, want a failure", err)
	}
}

func TestLoadRejectsAMissingDirectory(t *testing.T) {
	t.Parallel()

	if _, err := replay.Load(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("Load of a missing directory: want an error")
	}
}

// liveRegistries returns fakes of the live registries: exemple.fr is
// registered, exemple.com unknown, exemple.io not covered; 120027016 has a
// notice, 552100554 is unknown to SIRENE and has none.
func liveRegistries() app.Registries {
	return app.Registries{
		Domains: &liveDomains{answers: map[string]error{
			"exemple.fr": nil, "exemple.com": fmt.Errorf("rdap: %w", app.ErrNotFound),
			"exemple.io": fmt.Errorf("rdap: %w", app.ErrNotCovered),
		}},
		Companies: liveCompanies{},
		Notices:   liveNotices{},
	}
}

type liveDomains struct {
	answers map[string]error
	err     error
}

func (*liveDomains) Source() domain.Source { return "RDAP" }

func (l *liveDomains) Lookup(_ context.Context, d domain.Domain) (domain.Registration, error) {
	if l.err != nil {
		return domain.Registration{}, l.err
	}
	if err := l.answers[d.Registrable()]; err != nil {
		return domain.Registration{}, err
	}
	return registered, nil
}

type liveCompanies struct{}

func (liveCompanies) Source() domain.Source { return "SIRENE" }

func (liveCompanies) Lookup(_ context.Context, n domain.SIREN) (domain.Company, error) {
	if n.String() == "120027016" {
		return company, nil
	}
	return domain.Company{}, fmt.Errorf("sirene: %w", app.ErrNotFound)
}

type liveNotices struct{}

func (liveNotices) Source() domain.Source { return "BODACC" }

func (liveNotices) Notices(_ context.Context, n domain.SIREN) ([]domain.Notice, error) {
	if n.String() == "120027016" {
		return notices, nil
	}
	return nil, nil
}

// mustLookup asks the registries about each input, as a Check would.
func mustLookup(t *testing.T, r app.Registries, ins ...string) {
	t.Helper()
	ctx := context.Background()
	for _, in := range ins {
		if n, err := domain.ParseSIREN(in); err == nil {
			_, _ = r.Companies.Lookup(ctx, n)
			_, _ = r.Notices.Notices(ctx, n)
			continue
		}
		_, _ = r.Domains.Lookup(ctx, mustDomain(t, in))
	}
}

func mustDomain(t *testing.T, s string) domain.Domain {
	t.Helper()
	d, err := domain.ParseDomain(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustSIREN(t *testing.T, s string) domain.SIREN {
	t.Helper()
	n, err := domain.ParseSIREN(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
