// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Registries are the sources a Check consults: the blacklists, then the
// registries. A nil registry is left out of the reports.
type Registries struct {
	Blacklists []Blacklist
	Domains    DomainRegistry
	Companies  CompanyRegistry
	Notices    NoticeRegistry
}

// Without returns r less the registries leave reports true for.
func (r Registries) Without(leave func(domain.Source) bool) Registries {
	r.Blacklists = slices.DeleteFunc(slices.Clone(r.Blacklists), func(b Blacklist) bool {
		return leave(b.Rule.Source())
	})
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

// Check reads the blacklists and consults the registries, all at once,
// about an input, and reports the signals they give and the state of each. It is safe for concurrent use when the
// registries are.
type Check struct {
	registries Registries
	timeout    time.Duration
}

// CheckOption configures a Check.
type CheckOption func(*Check)

// WithTimeout bounds every check to d, or to the deadline of its context
// when that comes first.
func WithTimeout(d time.Duration) CheckOption {
	return func(c *Check) { c.timeout = d }
}

// NewCheck returns a Check of the registries.
func NewCheck(r Registries, opts ...CheckOption) *Check {
	c := &Check{registries: r}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check returns the report of in. A blacklist without a snapshot in
// service, or a registry that fails or has not answered when the deadline
// comes, is reported unavailable; the others are still reported.
func (c *Check) Check(ctx context.Context, in domain.Input) domain.Report {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	// One slice holds every state: the hot path allocates it once.
	lookups := c.lookups(in)
	states := make([]domain.SourceState, 0, len(c.registries.Blacklists)+len(lookups))
	signals, states := readBlacklists(c.registries.Blacklists, in, states)
	return domain.NewReport(in.String(), signals, collect(ctx, lookups, states))
}

// lookup is the consultation of one registry about an input; run is nil
// when the registry does not cover the input.
type lookup struct {
	source domain.Source
	run    func(ctx context.Context) error
}

// lookups returns the consultation of each registry about in, in report
// order.
func (c *Check) lookups(in domain.Input) []lookup {
	r := c.registries
	d, isDomain := domainOf(in)
	n, isCompany := companyOf(in)
	out := make([]lookup, 0, 3)
	if r.Domains != nil {
		l := lookup{source: r.Domains.Source()}
		if isDomain {
			l.run = func(ctx context.Context) error {
				_, err := r.Domains.Lookup(ctx, d)
				return err
			}
		}
		out = append(out, l)
	}
	if r.Companies != nil {
		l := lookup{source: r.Companies.Source()}
		if isCompany {
			l.run = func(ctx context.Context) error {
				_, err := r.Companies.Lookup(ctx, n)
				return err
			}
		}
		out = append(out, l)
	}
	if r.Notices != nil {
		l := lookup{source: r.Notices.Source()}
		if isCompany {
			l.run = func(ctx context.Context) error {
				_, err := r.Notices.Notices(ctx, n)
				return err
			}
		}
		out = append(out, l)
	}
	return out
}

// answer is the outcome of the lookup whose state is at index i.
type answer struct {
	i   int
	err error
}

// collect runs the lookups that apply concurrently and appends the state of
// each to states, in order. It returns once all answered or ctx is done: a
// lookup still running then is reported unavailable and its answer
// discarded.
func collect(ctx context.Context, lookups []lookup, states []domain.SourceState) []domain.SourceState {
	base := len(states)
	for _, l := range lookups {
		states = append(states, domain.SourceState{Source: l.source, Status: domain.SourceNotApplicable})
	}
	var answers chan answer
	pending := 0
	for i, l := range lookups {
		if l.run == nil {
			continue
		}
		if answers == nil {
			answers = make(chan answer, len(lookups))
		}
		states[base+i].Status = domain.SourceUnavailable
		pending++
		go func() { answers <- answer{base + i, l.run(ctx)} }()
	}
	for ; pending > 0; pending-- {
		select {
		case a := <-answers:
			states[a.i].Status = statusOf(a.err)
		case <-ctx.Done():
			return states
		}
	}
	return states
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
