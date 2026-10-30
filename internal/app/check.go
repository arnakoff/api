// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// Registries are the sources a Check consults: the blacklists, the white
// lists, then the registries. A nil registry is left out of the reports.
type Registries struct {
	Blacklists []Blacklist
	Whitelists []Whitelist
	Domains    DomainRegistry
	Companies  CompanyRegistry
	Notices    NoticeRegistry
}

// Without returns r less the registries leave reports true for.
func (r Registries) Without(leave func(domain.Source) bool) Registries {
	r.Blacklists = slices.DeleteFunc(slices.Clone(r.Blacklists), func(b Blacklist) bool {
		return leave(b.Rule.Source())
	})
	r.Whitelists = slices.DeleteFunc(slices.Clone(r.Whitelists), func(w Whitelist) bool {
		return leave(w.Rule.Source())
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

// Check reads the blacklists and white lists and consults the registries,
// all at once, about an input, and reports the signals they give and the
// state of each. It is safe for concurrent use when the registries are.
type Check struct {
	registries Registries
	clock      Clock
	timeout    time.Duration
}

// CheckOption configures a Check.
type CheckOption func(*Check)

// WithTimeout bounds every check to d, or to the deadline of its context
// when that comes first.
func WithTimeout(d time.Duration) CheckOption {
	return func(c *Check) { c.timeout = d }
}

// NewCheck returns a Check of the registries, which dates its findings
// with clock.
func NewCheck(r Registries, clock Clock, opts ...CheckOption) *Check {
	c := &Check{registries: r, clock: clock}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check returns the report of in. A list without a snapshot in service,
// or a registry that fails or has not answered when the deadline comes, is
// reported unavailable; the others are still reported.
func (c *Check) Check(ctx context.Context, in domain.Input) domain.Report {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	// One slice holds every state: the hot path allocates it once.
	lookups := c.lookups(in)
	r := c.registries
	states := make([]domain.SourceState, 0, len(r.Blacklists)+len(r.Whitelists)+len(lookups))
	signals, states := readLists(r.Blacklists, in, nil, states)
	signals, states = readLists(r.Whitelists, in, signals, states)
	signals, states = collect(ctx, lookups, signals, states)
	return domain.NewReport(in.String(), signals, states)
}

// lookup is the consultation of one registry about an input; run is nil
// when the registry does not cover the input. run returns the signal the
// answer gives, or nil.
type lookup struct {
	source domain.Source
	run    func(ctx context.Context) (*domain.Signal, error)
}

// lookups returns the consultation of each registry about in, in report
// order.
func (c *Check) lookups(in domain.Input) []lookup {
	r := c.registries
	out := make([]lookup, 0, 3)
	if r.Domains != nil {
		out = append(out, c.domainLookup(in))
	}
	if r.Companies != nil {
		out = append(out, companyLookup(r.Companies, in))
	}
	if r.Notices != nil {
		out = append(out, noticeLookup(r.Notices, in))
	}
	return out
}

// domainLookup returns the consultation of the domain registry about in,
// whose answer gives domain_recent.
func (c *Check) domainLookup(in domain.Input) lookup {
	reg := c.registries.Domains
	source := reg.Source()
	d, ok := domainOf(in)
	if !ok {
		return lookup{source: source}
	}
	return lookup{source: source, run: func(ctx context.Context) (*domain.Signal, error) {
		rec, err := reg.Lookup(ctx, d)
		if err != nil {
			return nil, err
		}
		if s, found := signal.DomainRecent(d, rec, source, c.clock.Now()); found {
			return &s, nil
		}
		return nil, nil
	}}
}

// companyLookup returns the consultation of the company register about in.
func companyLookup(reg CompanyRegistry, in domain.Input) lookup {
	l := lookup{source: reg.Source()}
	if n, ok := companyOf(in); ok {
		l.run = func(ctx context.Context) (*domain.Signal, error) {
			_, err := reg.Lookup(ctx, n)
			return nil, err
		}
	}
	return l
}

// noticeLookup returns the consultation of the legal notices about in.
func noticeLookup(reg NoticeRegistry, in domain.Input) lookup {
	l := lookup{source: reg.Source()}
	if n, ok := companyOf(in); ok {
		l.run = func(ctx context.Context) (*domain.Signal, error) {
			_, err := reg.Notices(ctx, n)
			return nil, err
		}
	}
	return l
}

// answer is the outcome of the lookup whose state is at index i, and the
// signal it gives, or nil.
type answer struct {
	i      int
	signal *domain.Signal
	err    error
}

// collect runs the lookups that apply concurrently, appends the state of
// each to states, in order, and the signals they give to signals, in the
// same order. It returns once all answered or ctx is done: a lookup still
// running then is reported unavailable and its answer discarded.
func collect(ctx context.Context, lookups []lookup, signals []domain.Signal, states []domain.SourceState) ([]domain.Signal, []domain.SourceState) {
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
		go func() {
			s, err := l.run(ctx)
			answers <- answer{base + i, s, err}
		}()
	}
	var found []answer
	for ; pending > 0; pending-- {
		select {
		case a := <-answers:
			states[a.i].Status = statusOf(a.err)
			if a.signal != nil {
				found = append(found, a)
			}
		case <-ctx.Done():
			return appendFound(signals, found), states
		}
	}
	return appendFound(signals, found), states
}

// appendFound appends the signals of found to signals in lookup order,
// whatever order the answers came in.
func appendFound(signals []domain.Signal, found []answer) []domain.Signal {
	slices.SortFunc(found, func(a, b answer) int { return a.i - b.i })
	for _, a := range found {
		signals = append(signals, *a.signal)
	}
	return signals
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
