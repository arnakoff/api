// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/arnakoff/api/internal/adapter/alert"
	"github.com/arnakoff/api/internal/adapter/mcp"
	"github.com/arnakoff/api/internal/adapter/reference/hosting"
	"github.com/arnakoff/api/internal/adapter/registry/live"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/startup"
)

// reloadEvery is how often the snapshots stored by the sync process are
// looked for: a new snapshot is in service within this delay.
const reloadEvery = time.Minute

// checkTimeout bounds a check, registries included: a slow registry is
// reported unavailable instead of holding the assistant.
const checkTimeout = 10 * time.Second

func main() {
	// Logs go to stderr: stdout carries the protocol.
	os.Exit(startup.Run("mcp", os.LookupEnv, os.Stderr, run))
}

// run is the work of the command, once its configuration is loaded and the
// redacting logger is the slog default: the MCP server on stdin and
// stdout, until the client leaves.
func run(ctx context.Context, cfg config.Config) error {
	disk, err := store.NewDisk(cfg.SnapshotDir())
	if err != nil {
		return err
	}
	lists := catalog.Options{ACPR: cfg.SourceACPR}
	cat := catalog.New(catalog.Offline{}, lists)
	s, err := app.NewSync(disk, clock.System{}, cat.Fetchers()...)
	if err != nil {
		return err
	}
	check, err := newCheck(lists, s)
	if err != nil {
		return err
	}
	srcs, err := app.NewSourceList(s, cat.Infos(), append(live.Infos(), signal.IBANInfo())...)
	if err != nil {
		return err
	}
	if err := s.Restore(ctx); err != nil {
		slog.ErrorContext(ctx, "stored snapshot not restored", "error", err.Error())
	}
	// The sync process posts to the webhook; this one only logs.
	r := app.NewReloader(s, clock.System{}, alert.LogOnly(slog.Default()), reloadEvery)
	return serve(ctx, r, mcp.NewServer(mcp.Options{Check: check, Sources: srcs}), mcp.Stdio())
}

// runner runs until its context is done.
type runner interface {
	Run(ctx context.Context)
}

// serve runs srv on t and r beside it, until the client leaves or ctx is
// done.
func serve(ctx context.Context, r runner, srv *sdk.Server, t sdk.Transport) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		r.Run(ctx)
		return nil
	})
	g.Go(func() error {
		defer cancel()
		return srv.Run(ctx, t)
	})
	return g.Wait()
}

// newCheck returns the check of the lists opts turns on, in service in
// snaps, and of the live registries, each check bounded by checkTimeout.
func newCheck(opts catalog.Options, snaps listing.Snapshots) (*app.Check, error) {
	platforms, err := hosting.Load()
	if err != nil {
		return nil, err
	}
	regs, err := live.Registries()
	if err != nil {
		return nil, err
	}
	lists := catalog.Lists(opts, snaps, platforms)
	regs.Blacklists, regs.Whitelists, regs.Numbering = lists.Blacklists, lists.Whitelists, lists.Numbering
	return app.NewCheck(regs, clock.System{}, app.WithTimeout(checkTimeout)), nil
}
