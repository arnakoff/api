// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"os"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/startup"
)

// exitUsage is the exit code of a misuse of the arguments.
const exitUsage = 2

func main() {
	cmd, err := parseArgs(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		os.Exit(exitUsage)
	}
	os.Exit(startup.Run("admin", os.LookupEnv, os.Stderr, func(ctx context.Context, cfg config.Config) error {
		return run(ctx, cfg, cmd)
	}))
}

// run is the work of the command, once its configuration is loaded and the
// redacting logger is the slog default.
func run(ctx context.Context, cfg config.Config, cmd command) error {
	if cmd.action == actionMigrate {
		if err := cfg.RequireDatabase(); err != nil {
			return err
		}
		return postgres.Migrate(ctx, string(cfg.DatabaseURL.Bytes()))
	}
	if err := cfg.RequireKeyStore(); err != nil {
		return err
	}
	keys, err := postgres.Open(ctx, string(cfg.DatabaseURL.Bytes()))
	if err != nil {
		return err
	}
	defer keys.Close()
	admin := app.NewKeyAdmin(keys, cfg.APIKeySecret.Bytes(), clock.System{}, rand.Reader)
	return runKey(ctx, admin, cfg, cmd)
}

// runKey runs a key action with admin, writing to standard output.
func runKey(ctx context.Context, admin *app.KeyAdmin, cfg config.Config, cmd command) error {
	switch cmd.action {
	case actionCreate:
		env, err := keyEnv(cmd.env, cfg.APIKeyEnv)
		if err != nil {
			return err
		}
		key, rec, err := admin.Issue(ctx, env, cmd.label)
		if err != nil {
			return err
		}
		return writeIssued(os.Stdout, key, rec)
	case actionList:
		recs, err := admin.List(ctx)
		if err != nil {
			return err
		}
		return writeKeys(os.Stdout, recs)
	default:
		return admin.Revoke(ctx, cmd.id)
	}
}

// keyEnv returns the environment flag names, or the configured one when
// the flag is empty.
func keyEnv(flagEnv, configured string) (domain.KeyEnvironment, error) {
	if flagEnv == "" {
		flagEnv = configured
	}
	return domain.ParseKeyEnvironment(flagEnv)
}
