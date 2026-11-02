// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/arnakoff/api/internal/adapter/alert"
	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/startup"
)

// reloadEvery is how often the snapshots stored by the sync process are
// looked for: a new snapshot is in service within this delay.
const reloadEvery = time.Minute

func main() {
	os.Exit(startup.Run("api", os.LookupEnv, os.Stderr, run))
}

// run is the work of the command, once its configuration is loaded and the
// redacting logger is the slog default.
func run(ctx context.Context, cfg config.Config) error {
	disk, err := store.NewDisk(cfg.SnapshotDir())
	if err != nil {
		return err
	}
	cat := catalog.New(catalog.Offline{}, catalog.Options{ACPR: cfg.SourceACPR})
	s, err := app.NewSync(disk, clock.System{}, cat.Fetchers()...)
	if err != nil {
		return err
	}
	if err := s.Restore(ctx); err != nil {
		slog.ErrorContext(ctx, "stored snapshot not restored", "error", err.Error())
	}
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.Addr)
	if err != nil {
		return err
	}
	logger := slog.Default()
	opts := httpapi.Options{Ready: s, Logger: logger, ProblemBase: cfg.ProblemBaseURL}
	srv := httpapi.NewServer(httpapi.NewHandler(opts), logger)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		// The sync process posts to the webhook; this one only logs.
		app.NewReloader(s, clock.System{}, alert.LogOnly(logger), reloadEvery).Run(ctx)
		return nil
	})
	g.Go(func() error { return httpapi.Serve(ctx, srv, ln) })
	return g.Wait()
}
