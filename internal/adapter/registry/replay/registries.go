// SPDX-License-Identifier: AGPL-3.0-only

package replay

import (
	"context"
	"time"

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

// Replayed returns registries that answer from the tape alone, each answer
// after latency: a stand-in for the live registries under load.
func (t *Tape) Replayed(latency time.Duration) app.Registries {
	return app.Registries{
		Domains:   &domains{tape: t, latency: latency},
		Companies: &companies{tape: t, latency: latency},
		Notices:   &notices{tape: t, latency: latency},
	}
}

// wait returns after d, or with the error of ctx if it is done first.
func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// domains asks live and records its answers, or replays them after latency
// when live is nil.
type domains struct {
	tape    *Tape
	live    app.DomainRegistry
	latency time.Duration
}

// Source names the registry.
func (*domains) Source() domain.Source { return rdap.Source }

// Lookup returns the registration of n's registrable domain.
func (d *domains) Lookup(ctx context.Context, n domain.Domain) (domain.Registration, error) {
	key := n.Registrable()
	if d.live == nil {
		if err := wait(ctx, d.latency); err != nil {
			return domain.Registration{}, err
		}
		r, err := d.tape.domains.play(key)
		return domain.Registration(r), err
	}
	r, err := d.live.Lookup(ctx, n)
	d.tape.domains.record(key, registration(r), err)
	return r, err
}

// companies asks live and records its answers, or replays them after
// latency when live is nil.
type companies struct {
	tape    *Tape
	live    app.CompanyRegistry
	latency time.Duration
}

// Source names the registry.
func (*companies) Source() domain.Source { return sirene.Source }

// Lookup returns the company n identifies.
func (c *companies) Lookup(ctx context.Context, n domain.SIREN) (domain.Company, error) {
	key := n.String()
	if c.live == nil {
		if err := wait(ctx, c.latency); err != nil {
			return domain.Company{}, err
		}
		v, err := c.tape.companies.play(key)
		return domain.Company(v), err
	}
	v, err := c.live.Lookup(ctx, n)
	c.tape.companies.record(key, company(v), err)
	return v, err
}

// notices asks live and records its answers, or replays them after latency
// when live is nil.
type notices struct {
	tape    *Tape
	live    app.NoticeRegistry
	latency time.Duration
}

// Source names the registry.
func (*notices) Source() domain.Source { return bodacc.Source }

// Notices returns the notices about the company n identifies.
func (s *notices) Notices(ctx context.Context, n domain.SIREN) ([]domain.Notice, error) {
	key := n.String()
	if s.live == nil {
		if err := wait(ctx, s.latency); err != nil {
			return nil, err
		}
		v, err := s.tape.notices.play(key)
		return toNotices(v), err
	}
	v, err := s.live.Notices(ctx, n)
	s.tape.notices.record(key, fromNotices(v), err)
	return v, err
}
