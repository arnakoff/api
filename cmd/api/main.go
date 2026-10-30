// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/arnakoff/api/internal/adapter/alert"
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
	// The sync process posts to the webhook; this one only logs.
	app.NewReloader(s, clock.System{}, alert.LogOnly(slog.Default()), reloadEvery).Run(ctx)
	return nil
}
