// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"log/slog"
	"net"

	"golang.org/x/sync/errgroup"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/app"
)

// serveMigration serves the probes of m on ln until ctx is done, and
// applies the migrations of m meanwhile. A failure is logged and leaves
// /readyz at 503: the process keeps serving, so that the platform shows it
// unready instead of restarting it in a loop.
func serveMigration(ctx context.Context, ln net.Listener, m *app.Migration, logger *slog.Logger) error {
	srv := httpapi.NewServer(httpapi.NewProbeHandler(m), logger)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return httpapi.Serve(ctx, srv, ln) })
	g.Go(func() error {
		if err := m.Run(ctx); err != nil {
			logger.ErrorContext(ctx, "migrations not applied", "error", err.Error())
			return nil
		}
		logger.InfoContext(ctx, "migrations applied")
		return nil
	})
	return g.Wait()
}
