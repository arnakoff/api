// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/app"
)

func TestMigrationIsReadyOnceApplied(t *testing.T) {
	t.Parallel()

	calls := 0
	m := app.NewMigration(app.MigrateFunc(func(context.Context) error { calls++; return nil }))
	if m.Ready() {
		t.Fatal("Ready() before Run = true, want false")
	}
	if err := m.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !m.Ready() || calls != 1 {
		t.Errorf("after Run: Ready() = %v, migrations run %d times; want true, 1", m.Ready(), calls)
	}
}

func TestMigrationIsNotReadyWhenItFails(t *testing.T) {
	t.Parallel()

	failed := errors.New("connection refused")
	m := app.NewMigration(app.MigrateFunc(func(context.Context) error { return failed }))
	if err := m.Run(context.Background()); !errors.Is(err, failed) {
		t.Fatalf("Run() error = %v, want %v", err, failed)
	}
	if m.Ready() {
		t.Error("Ready() after a failed Run = true, want false")
	}
}
