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

// serveOnce serves the probes of o on ln until ctx is done, and runs the
// step of o meanwhile, named step in the logs. A failure is logged and
// leaves /readyz at 503: the process keeps serving, so that the platform
// shows it unready instead of restarting it in a loop.
func serveOnce(ctx context.Context, ln net.Listener, step string, o *app.Once, logger *slog.Logger) error {
	srv := httpapi.NewServer(httpapi.NewProbeHandler(o), logger)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return httpapi.Serve(ctx, srv, ln) })
	g.Go(func() error {
		if err := o.Run(ctx); err != nil {
			logger.ErrorContext(ctx, "step failed", "step", step, "error", err.Error())
			return nil
		}
		logger.InfoContext(ctx, "step done", "step", step)
		return nil
	})
	return g.Wait()
}
