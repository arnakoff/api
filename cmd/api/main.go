// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/arnakoff/api/internal/adapter/alert"
	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/adapter/mcp"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/adapter/turnstile"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/buildinfo"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/ratelimit"
	"github.com/arnakoff/api/internal/platform/startup"
)

// reloadEvery is how often the snapshots stored by the sync process are
// looked for: a new snapshot is in service within this delay.
const reloadEvery = time.Minute

// checkTimeout bounds a check, registries included. It is shorter than the
// server's write timeout: a slow registry is reported unavailable instead
// of cutting the response.
const checkTimeout = 10 * time.Second

// exitUsage is the exit code of a misuse of the arguments.
const exitUsage = 2

func main() {
	cmd, err := parseArgs(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(exitUsage)
	}
	if cmd == probeCommand {
		os.Exit(startup.Run("api probe", os.LookupEnv, os.Stderr, probe))
	}
	os.Exit(startup.Run("api", os.LookupEnv, os.Stderr, run))
}

// probe exits 0 when the API of cfg answers /healthz: the health check of
// the container image, which has no shell.
func probe(ctx context.Context, cfg config.Config) error {
	return httpapi.ProbeAlive(ctx, cfg.Addr)
}

// run is the work of the command, once its configuration is loaded and the
// redacting logger is the slog default.
func run(ctx context.Context, cfg config.Config) error {
	if err := cfg.RequireTrustedProxies(); err != nil {
		return err
	}
	snaps, err := assembly.SnapshotStore(cfg)
	if err != nil {
		return err
	}
	svc, err := assembly.Live(catalog.Options{ACPR: cfg.SourceACPR}, snaps, app.WithTimeout(checkTimeout))
	if err != nil {
		return err
	}
	keys, err := postgres.OpenKeyServices(ctx, cfg)
	if err != nil {
		return err
	}
	defer keys.Close()
	if err := svc.Sync.Restore(ctx); err != nil {
		slog.ErrorContext(ctx, "stored snapshot not restored", "error", err.Error())
	}
	opts, err := withAccess(cfg, httpapi.Options{
		Ready: svc.Sync, Check: svc.Check, Auth: keys.Auth, Sources: svc.Sources, Logger: slog.Default(), ProblemBase: cfg.ProblemBaseURL,
		BatchMaxItems: cfg.BatchMaxItems, TrustedProxies: cfg.TrustedProxies, ClientIPHeader: cfg.ClientIPHeader,
		MCP: mcp.HTTP(mcp.NewServer(mcp.Options{Check: svc.Check, Sources: svc.Sources})),
	})
	if err != nil {
		return err
	}
	return serve(ctx, cfg, svc.Sync, keys.Purge, opts)
}

// withAccess returns opts with the access rules of cfg: its rate limits
// and its demo access.
func withAccess(cfg config.Config, opts httpapi.Options) (httpapi.Options, error) {
	demo, err := newDemo(cfg.Demo)
	if err != nil {
		return httpapi.Options{}, err
	}
	opts.KeyLimit = ratelimit.PerMinute(cfg.RateLimitKey, clock.System{}.Now)
	opts.IPLimit = ratelimit.PerMinute(cfg.RateLimitIP, clock.System{}.Now)
	opts.AuthLimit = ratelimit.PerMinute(cfg.RateLimitAuth, clock.System{}.Now)
	opts.Demo = demo
	return opts, nil
}

// serve answers the API of opts on cfg.Addr, reloads the snapshots of s
// and purges the revoked keys with purge until ctx is done.
func serve(ctx context.Context, cfg config.Config, s *app.Sync, purge *app.KeyPurge, opts httpapi.Options) error {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.Addr)
	if err != nil {
		return err
	}
	srv := httpapi.NewServer(httpapi.NewHandler(opts), opts.Logger)
	opts.Logger.InfoContext(ctx, "api listening", "addr", ln.Addr().String(), "version", buildinfo.Version())
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		// The sync process posts to the webhook; this one only logs.
		app.NewReloader(s, clock.System{}, alert.LogOnly(opts.Logger), reloadEvery).Run(ctx)
		return nil
	})
	g.Go(func() error {
		purge.Run(ctx, logPurge(opts.Logger))
		return nil
	})
	g.Go(func() error { return httpapi.Serve(ctx, srv, ln) })
	return g.Wait()
}

// logPurge returns the report of the key purge: a line on logger for each
// purge.
func logPurge(logger *slog.Logger) app.PurgeReport {
	return func(ctx context.Context, deleted int64, err error) {
		if err != nil {
			logger.ErrorContext(ctx, "revoked API keys not purged", "error", err.Error())
			return
		}
		logger.InfoContext(ctx, "revoked API keys purged", "deleted", deleted)
	}
}

// newDemo returns the demo access cfg configures: without a verifier when
// it is off, so that the route answers 503.
func newDemo(cfg config.Demo) (httpapi.DemoOptions, error) {
	opts := httpapi.DemoOptions{
		Origin:      cfg.Origin,
		Limit:       ratelimit.PerMinute(cfg.RateLimit, clock.System{}.Now),
		GlobalLimit: ratelimit.PerMinute(cfg.RateLimitGlobal, clock.System{}.Now),
	}
	if !cfg.Enabled() {
		return opts, nil
	}
	v, err := turnstile.NewLive(cfg.TurnstileSecret.Bytes(), cfg.Hostname())
	if err != nil {
		return httpapi.DemoOptions{}, err
	}
	opts.Verifier = v
	return opts, nil
}
