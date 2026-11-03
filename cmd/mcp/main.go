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
	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/mcp"
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
	svc, err := assembly.Live(catalog.Options{ACPR: cfg.SourceACPR}, disk, app.WithTimeout(checkTimeout))
	if err != nil {
		return err
	}
	if err := svc.Sync.Restore(ctx); err != nil {
		slog.ErrorContext(ctx, "stored snapshot not restored", "error", err.Error())
	}
	// The sync process posts to the webhook; this one only logs.
	r := app.NewReloader(svc.Sync, clock.System{}, alert.LogOnly(slog.Default()), reloadEvery)
	return serve(ctx, r, mcp.NewServer(mcp.Options{Check: svc.Check, Sources: svc.Sources}), mcp.Stdio())
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
