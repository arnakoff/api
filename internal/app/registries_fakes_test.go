// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// asked records what a fake registry was asked about and answers err,
// after hold returns when it is set.
type asked struct {
	mu   sync.Mutex
	ins  []string
	err  error
	hold func(ctx context.Context)
}

func (a *asked) record(ctx context.Context, in string) error {
	a.mu.Lock()
	a.ins = append(a.ins, in)
	hold, err := a.hold, a.err
	a.mu.Unlock()
	if hold != nil {
		hold(ctx)
	}
	return err
}

func (a *asked) got() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.ins)
}

type fakeDomains struct{ asked }

func (*fakeDomains) Source() domain.Source { return "RDAP" }

func (f *fakeDomains) Lookup(ctx context.Context, d domain.Domain) (domain.Registration, error) {
	return domain.Registration{}, f.record(ctx, d.String())
}

type fakeCompanies struct{ asked }

func (*fakeCompanies) Source() domain.Source { return "SIRENE" }

func (f *fakeCompanies) Lookup(ctx context.Context, n domain.SIREN) (domain.Company, error) {
	return domain.Company{}, f.record(ctx, n.String())
}

type fakeNotices struct{ asked }

func (*fakeNotices) Source() domain.Source { return "BODACC" }

func (f *fakeNotices) Notices(ctx context.Context, n domain.SIREN) ([]domain.Notice, error) {
	return nil, f.record(ctx, n.String())
}

// registries holds one fake of each registry.
type registries struct {
	domains   *fakeDomains
	companies *fakeCompanies
	notices   *fakeNotices
}

func newRegistries() registries {
	return registries{&fakeDomains{}, &fakeCompanies{}, &fakeNotices{}}
}

func (r registries) all() app.Registries {
	return app.Registries{Domains: r.domains, Companies: r.companies, Notices: r.notices}
}

// assertAsked checks the domain registry was asked about domains, and the
// company and notice registries about sirens.
func (r registries) assertAsked(t *testing.T, domains, sirens []string) {
	t.Helper()
	if got := r.domains.got(); !slices.Equal(got, domains) {
		t.Errorf("domain registry asked %v, want %v", got, domains)
	}
	if got := r.companies.got(); !slices.Equal(got, sirens) {
		t.Errorf("company registry asked %v, want %v", got, sirens)
	}
	if got := r.notices.got(); !slices.Equal(got, sirens) {
		t.Errorf("notice registry asked %v, want %v", got, sirens)
	}
}
