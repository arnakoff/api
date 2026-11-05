// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"log/slog"
	"net"
	"os"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/httpapi"
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
	switch cmd.action {
	case actionMigrate:
		return runMigrate(ctx, cfg, cmd.listen)
	case actionBackup:
		return runBackup(ctx, cfg, cmd.listen)
	case actionRestore:
		return runRestore(ctx, cfg, cmd.file)
	}
	if err := cfg.RequireKeyStore(); err != nil {
		return err
	}
	keys, err := postgres.Open(ctx, string(cfg.DatabaseURL.Bytes()), postgres.WithCA([]byte(cfg.DatabaseCA)))
	if err != nil {
		return err
	}
	defer keys.Close()
	admin := app.NewKeyAdmin(keys, cfg.APIKeySecret.Bytes(), clock.System{}, rand.Reader)
	return runKey(ctx, admin, cfg, cmd)
}

// runMigrate applies the migrations of the configured database, then
// exits, or serves its probes on listen when it is set.
func runMigrate(ctx context.Context, cfg config.Config, listen string) error {
	if err := cfg.RequireDatabase(); err != nil {
		return err
	}
	migrate := app.MigrateFunc(func(ctx context.Context) error {
		return postgres.Migrate(ctx, string(cfg.DatabaseURL.Bytes()), postgres.WithCA([]byte(cfg.DatabaseCA)))
	})
	if listen == "" {
		return migrate(ctx)
	}
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", listen)
	if err != nil {
		return err
	}
	return serveMigration(ctx, ln, app.NewMigration(migrate), slog.Default())
}

// runBackup copies the key store, sealed, to its bucket, then exits, or
// serves POST /backup on listen when it is set.
func runBackup(ctx context.Context, cfg config.Config, listen string) error {
	backup, closeBackup, err := assembly.Backup(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeBackup()
	if listen == "" {
		name, err := backup.Run(ctx)
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "backup kept", "object", name)
		return nil
	}
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", listen)
	if err != nil {
		return err
	}
	srv := httpapi.NewServer(httpapi.NewBackupHandler(backup, slog.Default()), slog.Default())
	return httpapi.Serve(ctx, srv, ln)
}

// runRestore loads the decrypted dump in file into the configured
// database.
func runRestore(ctx context.Context, cfg config.Config, file string) error {
	if err := cfg.RequireDatabase(); err != nil {
		return err
	}
	f, err := os.Open(file) //nolint:gosec // G304: the dump the operator names.
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // Read only.
	if err := postgres.Restore(ctx, string(cfg.DatabaseURL.Bytes()), f, postgres.WithCA([]byte(cfg.DatabaseCA))); err != nil {
		return err
	}
	slog.InfoContext(ctx, "dump restored and schema migrated")
	return nil
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
