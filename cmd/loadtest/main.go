// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/corpus"
	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/adapter/registry/replay"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/ratelimit"
	"github.com/arnakoff/api/internal/platform/startup"
)

// exitUsage is the exit code of a misuse of the flags.
const exitUsage = 2

// checkTimeout bounds a check, as in cmd/api.
const checkTimeout = 10 * time.Second

// pprofHeaderTimeout bounds the reading of a profile request's header. No
// write timeout: a CPU profile streams for the seconds it is asked for.
const pprofHeaderTimeout = 5 * time.Second

func main() {
	opts, err := parseFlags(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		os.Exit(exitUsage)
	}
	if opts.targets != "" {
		if err := targets(opts, os.Stdout); err != nil {
			_, _ = io.WriteString(os.Stderr, err.Error()+"\n")
			os.Exit(1)
		}
		return
	}
	os.Exit(startup.Run("loadtest", os.LookupEnv, os.Stderr, func(ctx context.Context, cfg config.Config) error {
		return run(ctx, opts, cfg)
	}))
}

// targets writes the vegeta targets of the corpus of opts to w, carrying
// the keys of opts.keys if it names a file.
func targets(opts options, w io.Writer) error {
	cases, err := corpus.Load(opts.corpus)
	if err != nil {
		return err
	}
	var keys []domain.APIKey
	if opts.keys != "" {
		if keys, err = loadKeys(opts.keys); err != nil {
			return err
		}
	}
	return writeTargets(w, opts.targets, cases, keys)
}

// loadKeys returns the API keys of the file at path.
func loadKeys(path string) ([]domain.APIKey, error) {
	f, err := os.Open(path) //nolint:gosec // G304: a path the operator names on the command line.
	if err != nil {
		return nil, fmt.Errorf("open keys: %w", err)
	}
	defer func() { _ = f.Close() }() // Read only: a close error changes nothing.
	return readKeys(f)
}

// run serves the API as cmd/api does, but with the registries replayed
// from the corpus tape after opts.latency, and the profiles on opts.pprof.
func run(ctx context.Context, opts options, cfg config.Config) error {
	if err := cfg.RequireTrustedProxies(); err != nil {
		return err
	}
	tape, err := replay.Load(filepath.Join(opts.corpus, "replay"))
	if err != nil {
		return err
	}
	svc, err := newService(ctx, cfg, tape.Replayed(opts.latency))
	if err != nil {
		return err
	}
	auth, closeKeys, err := postgres.OpenAuthenticator(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeKeys()
	return serve(ctx, cfg, opts.pprof, httpapi.Options{
		Ready: svc.Sync, Check: svc.Check, Auth: auth, Sources: svc.Sources, Logger: slog.Default(), ProblemBase: cfg.ProblemBaseURL,
		BatchMaxItems:  cfg.BatchMaxItems,
		KeyLimit:       ratelimit.PerMinute(cfg.RateLimitKey, clock.System{}.Now),
		IPLimit:        ratelimit.PerMinute(cfg.RateLimitIP, clock.System{}.Now),
		AuthLimit:      ratelimit.PerMinute(cfg.RateLimitAuth, clock.System{}.Now),
		TrustedProxies: cfg.TrustedProxies,
	})
}

// newService returns the service of the lists cfg turns on and of regs,
// each check bounded by checkTimeout, with the stored snapshots restored.
func newService(ctx context.Context, cfg config.Config, regs app.Registries) (assembly.Service, error) {
	disk, err := store.NewDisk(cfg.SnapshotDir())
	if err != nil {
		return assembly.Service{}, err
	}
	svc, err := assembly.New(catalog.Options{ACPR: cfg.SourceACPR}, disk, regs, app.WithTimeout(checkTimeout))
	if err != nil {
		return assembly.Service{}, err
	}
	if err := svc.Sync.Restore(ctx); err != nil {
		return assembly.Service{}, err
	}
	return svc, nil
}

// serve answers the API of opts on cfg.Addr, and the profiles on
// pprofAddr unless it is empty, until ctx is done.
func serve(ctx context.Context, cfg config.Config, pprofAddr string, opts httpapi.Options) error {
	g, ctx := errgroup.WithContext(ctx)
	if pprofAddr != "" {
		g.Go(func() error { return listenAndServe(ctx, pprofAddr, profiles()) })
	}
	g.Go(func() error {
		return listenAndServe(ctx, cfg.Addr, httpapi.NewServer(httpapi.NewHandler(opts), opts.Logger))
	})
	return g.Wait()
}

// listenAndServe serves srv on addr until ctx is done.
func listenAndServe(ctx context.Context, addr string, srv *http.Server) error {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return httpapi.Serve(ctx, srv, ln)
}

// profiles returns the server of the runtime profiles under /debug/pprof/.
func profiles() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	return &http.Server{Handler: mux, ReadHeaderTimeout: pprofHeaderTimeout}
}
