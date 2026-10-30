// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/arnakoff/api/internal/adapter/alert"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/startup"
)

// exitUsage is the exit code of a misuse of the flags.
const exitUsage = 2

func main() {
	opts, err := parseFlags(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		os.Exit(exitUsage)
	}
	os.Exit(startup.Run("sync", os.LookupEnv, os.Stderr, func(ctx context.Context, cfg config.Config) error {
		return run(ctx, cfg, opts)
	}))
}

// run is the work of the command, once its configuration is loaded and the
// redacting logger is the slog default.
func run(ctx context.Context, cfg config.Config, opts options) error {
	s, plans, err := newSync(cfg, opts.dryRun)
	if err != nil {
		return err
	}
	if err := s.Restore(ctx); err != nil {
		slog.ErrorContext(ctx, "stored snapshot not restored, syncing anyway", "error", err.Error())
	}
	alerter, err := newAlerter(cfg, opts.dryRun)
	if err != nil {
		return err
	}
	sc, err := app.NewScheduler(s, clock.System{}, alerter, clock.Jitter, plans...)
	if err != nil {
		return err
	}
	if !opts.once {
		slog.InfoContext(ctx, "sync scheduled", "sources", len(plans))
		sc.Run(ctx)
		return nil
	}
	if opts.only != "" {
		err = sc.Once(ctx, opts.only)
	} else {
		err = sc.Once(ctx)
	}
	report(ctx, s, opts.only)
	return err
}

// newSync wires the sources the configuration turns on to the snapshot
// store, or to a store that keeps nothing for a dry run.
func newSync(cfg config.Config, dryRun bool) (*app.Sync, []app.Plan, error) {
	sources := catalog.Options{ACPR: cfg.SourceACPR}
	client, err := httpclient.New(httpclient.Config{AllowedHosts: catalog.Hosts(sources)})
	if err != nil {
		return nil, nil, err
	}
	disk, err := store.NewDisk(cfg.SnapshotDir())
	if err != nil {
		return nil, nil, err
	}
	var st app.SnapshotStore = disk
	if dryRun {
		st = app.DryRun(disk)
	}
	cat := catalog.New(client, sources)
	s, err := app.NewSync(st, clock.System{}, cat.Fetchers()...)
	return s, cat.Plans(), err
}

// newAlerter returns an alerter that logs, and posts to the configured
// webhook unless this is a dry run.
func newAlerter(cfg config.Config, dryRun bool) (*alert.Notifier, error) {
	if cfg.AlertWebhook == nil || dryRun {
		return alert.LogOnly(slog.Default()), nil
	}
	return alert.ForWebhook(slog.Default(), string(cfg.AlertWebhook.Bytes()))
}

// report logs the state of the sources synced: only, or all when it is
// empty.
func report(ctx context.Context, s *app.Sync, only domain.Source) {
	for _, st := range s.States() {
		if only != "" && st.Source != only {
			continue
		}
		attrs := []any{"source", string(st.Source), "rows", st.Rows, "newest", st.Newest, "last_success", st.LastSuccess}
		if st.LastError != nil {
			attrs = append(attrs, "last_failure", st.LastFailure)
		}
		slog.InfoContext(ctx, "source state", attrs...)
	}
}
