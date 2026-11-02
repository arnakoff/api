// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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
func Migrate(ctx context.Context, conn string) error {
	cfg, err := ParseConfig(conn)
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*cfg.ConnConfig)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("migrate: connect: %w", err)
	}
	m, err := newMigrate(db)
	if err != nil {
		_ = db.Close()
		return err
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

// newMigrate returns a migrate.Migrate applying the embedded migrations
// through db, which its Close closes.
func newMigrate(db *sql.DB) (*migrate.Migrate, error) {
	drv, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return nil, fmt.Errorf("migrate: database driver: %w", err)
	}
	src, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: embedded migrations: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", drv)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return m, nil
}
