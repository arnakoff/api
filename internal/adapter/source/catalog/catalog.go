// SPDX-License-Identifier: AGPL-3.0-only

// Package catalog lists the sources Arnakoff syncs, how often, and the
// hosts they reach: the one list the sync and API processes are built from.
package catalog

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/adapter/source/acpr"
	"github.com/arnakoff/api/internal/adapter/source/amf"
	"github.com/arnakoff/api/internal/adapter/source/arcep"
	"github.com/arnakoff/api/internal/app"
)

// Sync intervals, chosen from the freshness of each source
// (docs/sources.md): a few times a day for the lists updated daily or at an
// unannounced pace, once a day for the weekly ARCEP files.
const (
	everyDaily  = 6 * time.Hour
	everyWeekly = 24 * time.Hour
)

// ErrOffline is returned by Offline.Get.
var ErrOffline = errors.New("this process downloads nothing")

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Offline is the Getter of a process that only restores snapshots, such as
// the API: it refuses every download.
type Offline struct{}

// Get returns ErrOffline.
func (Offline) Get(context.Context, string) ([]byte, error) {
	return nil, ErrOffline
}

// Options are the sources the configuration turns on beyond the default
// ones.
type Options struct {
	// ACPR turns the ACPR blacklist on, once its reuse licence is obtained.
	ACPR bool
}

// source is one row of the table: how to build its fetcher, the hosts it
// reaches, how often it is synced and whether Options turn it on.
type source struct {
	fetcher func(Getter) app.SourceFetcher
	hosts   func() []string
	every   time.Duration
	on      func(Options) bool
}

func always(Options) bool      { return true }
func acprOn(opts Options) bool { return opts.ACPR }

// sources is the one list of sources, in a stable order.
var sources = []source{
	{func(c Getter) app.SourceFetcher { return amf.NewBlacklist(c, amf.BlacklistDatasetURL) }, amf.Hosts, everyDaily, always},
	{func(c Getter) app.SourceFetcher { return amf.NewPSAN(c, amf.PSANDatasetURL) }, amf.Hosts, everyDaily, always},
	{func(c Getter) app.SourceFetcher { return amf.NewOffers(c, amf.OffersDatasetURL) }, amf.Hosts, everyDaily, always},
	{func(c Getter) app.SourceFetcher { return acpr.NewBlacklist(c, acpr.BlacklistURL) }, acpr.Hosts, everyDaily, acprOn},
	{func(c Getter) app.SourceFetcher {
		return arcep.NewNumbering(c, arcep.NumberingURL, arcep.OperatorsURL)
	}, arcep.Hosts, everyWeekly, always},
}

// Catalog is the list of the sources turned on, in a stable order.
type Catalog struct {
	entries []entry
}

type entry struct {
	fetcher app.SourceFetcher
	every   time.Duration
}

// New returns the catalog of the sources opts turns on, downloading through
// client.
func New(client Getter, opts Options) *Catalog {
	c := &Catalog{}
	for _, src := range sources {
		if src.on(opts) {
			c.entries = append(c.entries, entry{src.fetcher(client), src.every})
		}
	}
	return c
}

// Hosts returns, once each, the hosts the sources opts turns on reach, for
// the allowlist of the HTTP client New takes.
func Hosts(opts Options) []string {
	var hosts []string
	for _, src := range sources {
		if !src.on(opts) {
			continue
		}
		for _, h := range src.hosts() {
			if !slices.Contains(hosts, h) {
				hosts = append(hosts, h)
			}
		}
	}
	return hosts
}

// Fetchers returns the fetcher of each source, for app.NewSync.
func (c *Catalog) Fetchers() []app.SourceFetcher {
	fs := make([]app.SourceFetcher, len(c.entries))
	for i, e := range c.entries {
		fs[i] = e.fetcher
	}
	return fs
}

// Plans returns how often each source is synced, for app.NewScheduler.
func (c *Catalog) Plans() []app.Plan {
	ps := make([]app.Plan, len(c.entries))
	for i, e := range c.entries {
		ps[i] = app.Plan{Source: e.fetcher.Spec().Source, Every: e.every}
	}
	return ps
}
