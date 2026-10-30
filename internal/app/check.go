// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"

	"github.com/arnakoff/api/internal/domain"
)

// Registries are the registries a Check consults; a nil one is left out of
// the reports.
type Registries struct {
	Domains   DomainRegistry
	Companies CompanyRegistry
	Notices   NoticeRegistry
}

// Without returns r less the registries leave reports true for.
func (r Registries) Without(leave func(domain.Source) bool) Registries {
	if r.Domains != nil && leave(r.Domains.Source()) {
		r.Domains = nil
	}
	if r.Companies != nil && leave(r.Companies.Source()) {
		r.Companies = nil
	}
	if r.Notices != nil && leave(r.Notices.Source()) {
		r.Notices = nil
	}
	return r
}

// Check consults the registries that cover an input, one after the other,
// and reports the state of each. It is safe for concurrent use when the
// registries are.
type Check struct {
	registries Registries
}

// NewCheck returns a Check of the registries.
func NewCheck(r Registries) *Check {
	return &Check{registries: r}
}

// Check returns the report of in. A registry that fails is reported
// unavailable; the others are still consulted.
func (c *Check) Check(ctx context.Context, in domain.Input) domain.Report {
	r := c.registries
	d, isDomain := domainOf(in)
	n, isCompany := companyOf(in)
	states := make([]domain.SourceState, 0, 3)
	if r.Domains != nil {
		states = append(states, consult(r.Domains.Source(), isDomain, func() error {
			_, err := r.Domains.Lookup(ctx, d)
			return err
		}))
	}
	if r.Companies != nil {
		states = append(states, consult(r.Companies.Source(), isCompany, func() error {
			_, err := r.Companies.Lookup(ctx, n)
			return err
		}))
	}
	if r.Notices != nil {
		states = append(states, consult(r.Notices.Source(), isCompany, func() error {
			_, err := r.Notices.Notices(ctx, n)
			return err
		}))
	}
	return domain.NewReport(in.String(), nil, states)
}

// domainOf returns the domain of in, its own or its email address's, and
// whether in has one.
func domainOf(in domain.Input) (domain.Domain, bool) {
	switch v := in.(type) {
	case domain.Domain:
		return v, true
	case domain.Email:
		return v.Domain(), true
	default:
		return domain.Domain{}, false
	}
}

// companyOf returns the SIREN of in, and whether in identifies a company.
func companyOf(in domain.Input) (domain.SIREN, bool) {
	switch v := in.(type) {
	case domain.SIREN:
		return v, true
	case domain.SIRET:
		return v.SIREN(), true
	default:
		return domain.SIREN{}, false
	}
}

// consult runs lookup when the registry applies to the input and returns
// the registry's state.
func consult(source domain.Source, applies bool, lookup func() error) domain.SourceState {
	if !applies {
		return domain.SourceState{Source: source, Status: domain.SourceNotApplicable}
	}
	return domain.SourceState{Source: source, Status: statusOf(lookup())}
}

// statusOf returns the source status of a registry's answer.
func statusOf(err error) domain.SourceStatus {
	switch {
	case err == nil, errors.Is(err, ErrNotFound):
		return domain.SourceOK
	case errors.Is(err, ErrNotCovered):
		return domain.SourceNotApplicable
	default:
		return domain.SourceUnavailable
	}
}
