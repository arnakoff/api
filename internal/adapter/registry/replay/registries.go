// SPDX-License-Identifier: AGPL-3.0-only

package replay

import (
	"context"

	"github.com/arnakoff/api/internal/adapter/registry/bodacc"
	"github.com/arnakoff/api/internal/adapter/registry/rdap"
	"github.com/arnakoff/api/internal/adapter/registry/sirene"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// Registries returns registries that ask the live ones and record their
// answers on the tape; a nil live registry is replayed from the tape
// instead.
func (t *Tape) Registries(live app.Registries) app.Registries {
	return app.Registries{
		Domains:   &domains{tape: t, live: live.Domains},
		Companies: &companies{tape: t, live: live.Companies},
		Notices:   &notices{tape: t, live: live.Notices},
	}
}

// domains asks live and records its answers, or replays them when live is
// nil.
type domains struct {
	tape *Tape
	live app.DomainRegistry
}

// Source names the registry.
func (*domains) Source() domain.Source { return rdap.Source }

// Lookup returns the registration of n's registrable domain.
func (d *domains) Lookup(ctx context.Context, n domain.Domain) (domain.Registration, error) {
	key := n.Registrable()
	if d.live == nil {
		r, err := d.tape.domains.play(key)
		return domain.Registration(r), err
	}
	r, err := d.live.Lookup(ctx, n)
	d.tape.domains.record(key, registration(r), err)
	return r, err
}

// companies asks live and records its answers, or replays them when live
// is nil.
type companies struct {
	tape *Tape
	live app.CompanyRegistry
}

// Source names the registry.
func (*companies) Source() domain.Source { return sirene.Source }

// Lookup returns the company n identifies.
func (c *companies) Lookup(ctx context.Context, n domain.SIREN) (domain.Company, error) {
	key := n.String()
	if c.live == nil {
		v, err := c.tape.companies.play(key)
		return domain.Company(v), err
	}
	v, err := c.live.Lookup(ctx, n)
	c.tape.companies.record(key, company(v), err)
	return v, err
}

// notices asks live and records its answers, or replays them when live is
// nil.
type notices struct {
	tape *Tape
	live app.NoticeRegistry
}

// Source names the registry.
func (*notices) Source() domain.Source { return bodacc.Source }

// Notices returns the notices about the company n identifies.
func (s *notices) Notices(ctx context.Context, n domain.SIREN) ([]domain.Notice, error) {
	key := n.String()
	if s.live == nil {
		v, err := s.tape.notices.play(key)
		return toNotices(v), err
	}
	v, err := s.live.Notices(ctx, n)
	s.tape.notices.record(key, fromNotices(v), err)
	return v, err
}
