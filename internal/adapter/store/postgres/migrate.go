// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// migrations are the versioned schema changes, <version>_<name>.up.sql and
// .down.sql, embedded in the binary.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies every migration not yet applied to the database of conn,
// one at a time under an advisory lock, so that two runs never interleave.
// It is run by the administration command with a role that may change the
// schema, never by the API on start (ADR 0004). Cancelling ctx stops it
// after the migration in progress.
func Migrate(ctx context.Context, conn string, opts ...Option) error {
	cfg, err := ParseConfig(conn, opts...)
	if err != nil {
		return err
	}
	m, err := openMigrate(ctx, cfg.ConnConfig)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	// Close closes db; the migrations are applied or not by then.
	defer func() { _, _ = m.Close() }()
	stop := context.AfterFunc(ctx, func() {
		select {
		case m.GracefulStop <- true:
		default:
		}
	})
	defer stop()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: %w", err)
	}
	return ctx.Err()
}

// openMigrate connects to the database of cc and returns a
// migrate.Migrate applying the embedded migrations through it, which its
// Close closes.
func openMigrate(ctx context.Context, cc *pgx.ConnConfig) (*migrate.Migrate, error) {
	db := stdlib.OpenDB(*cc)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect: %w", err)
	}
	drv, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database driver: %w", err)
	}
	src, err := migrationSource()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return migrate.NewWithInstance("iofs", src, "pgx5", drv)
}

// migrationSource returns the embedded migrations.
func migrationSource() (source.Driver, error) {
	src, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("embedded migrations: %w", err)
	}
	return src, nil
}

// latestVersion returns the version of the last embedded migration.
func latestVersion() (uint, error) {
	src, err := migrationSource()
	if err != nil {
		return 0, err
	}
	defer func() { _ = src.Close() }()
	v, err := src.First()
	if err != nil {
		return 0, fmt.Errorf("embedded migrations: %w", err)
	}
	for {
		next, err := src.Next(v)
		if errors.Is(err, fs.ErrNotExist) {
			return v, nil
		}
		if err != nil {
			return 0, fmt.Errorf("embedded migrations: %w", err)
		}
		v = next
	}
}
