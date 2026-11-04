// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Migrator applies the schema migrations not yet applied to the store.
type Migrator interface {
	Migrate(ctx context.Context) error
}

// MigrateFunc is a Migrator calling itself.
type MigrateFunc func(ctx context.Context) error

// Migrate calls f.
func (f MigrateFunc) Migrate(ctx context.Context) error { return f(ctx) }

// Migration applies the migrations of a store once and tells whether they
// are applied, for the readiness probe of the process that runs it inside
// the private network (#259).
type Migration struct {
	migrator Migrator
	applied  atomic.Bool
}

// NewMigration returns a Migration of migrator, not yet applied.
func NewMigration(migrator Migrator) *Migration {
	return &Migration{migrator: migrator}
}

// Run applies the migrations. Ready reports true once it succeeded.
func (m *Migration) Run(ctx context.Context) error {
	if err := m.migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	m.applied.Store(true)
	return nil
}

// Ready reports whether Run applied the migrations.
func (m *Migration) Ready() bool {
	return m.applied.Load()
}
