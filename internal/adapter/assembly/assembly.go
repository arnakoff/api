// SPDX-License-Identifier: AGPL-3.0-only

// Package assembly builds a check and the list of its sources from the
// lists the configuration turns on, the snapshots in service and the
// registries: the one assembly every command shares, so that none of them
// consults fewer sources than another without saying so.
package assembly

import (
	"github.com/arnakoff/api/internal/adapter/reference/hosting"
	"github.com/arnakoff/api/internal/adapter/registry/live"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/clock"
)

// Service is what a process answers with: the snapshots in service, the
// check that reads them and the list of the sources it consults.
type Service struct {
	Sync    *app.Sync
	Check   *app.Check
	Sources *app.SourceList
}

// Live returns the service of the lists opts turns on, their snapshots
// read from st, and of the live registries, each check configured by
// checkOpts. Nothing is fetched until a check asks.
func Live(opts catalog.Options, st app.SnapshotStore, checkOpts ...app.CheckOption) (Service, error) {
	regs, err := live.Registries()
	if err != nil {
		return Service{}, err
	}
	return New(opts, st, regs, checkOpts...)
}

// New returns the service of the lists opts turns on, their snapshots read
// from st, and of the registries of regs, each check configured by
// checkOpts. Nothing is restored: the caller restores Service.Sync.
func New(opts catalog.Options, st app.SnapshotStore, regs app.Registries, checkOpts ...app.CheckOption) (Service, error) {
	s, err := Sync(opts, st)
	if err != nil {
		return Service{}, err
	}
	regs, err = Lists(opts, s, regs)
	if err != nil {
		return Service{}, err
	}
	srcs, err := app.NewSourceList(s, offline(opts).Infos(), append(live.Infos(), signal.IBANInfo())...)
	if err != nil {
		return Service{}, err
	}
	return Service{Sync: s, Check: app.NewCheck(regs, clock.System{}, checkOpts...), Sources: srcs}, nil
}

// Sync returns the sync of the sources opts turns on, stored in st. It
// never downloads: the sync process does.
func Sync(opts catalog.Options, st app.SnapshotStore) (*app.Sync, error) {
	return app.NewSync(st, clock.System{}, offline(opts).Fetchers()...)
}

// Lists returns regs with the lists a check reads, each from the snapshot
// of its source in snaps: the blacklists opts turns on, the white lists,
// which leave out the shipped hosting platforms, and the numbering plan.
// The registries of regs are kept.
func Lists(opts catalog.Options, snaps listing.Snapshots, regs app.Registries) (app.Registries, error) {
	platforms, err := hosting.Load()
	if err != nil {
		return app.Registries{}, err
	}
	lists := catalog.Lists(opts, snaps, platforms)
	regs.Blacklists, regs.Whitelists, regs.Numbering = lists.Blacklists, lists.Whitelists, lists.Numbering
	return regs, nil
}

// offline returns the catalog of the sources opts turns on, refusing every
// download.
func offline(opts catalog.Options) *catalog.Catalog {
	return catalog.New(catalog.Offline{}, opts)
}
