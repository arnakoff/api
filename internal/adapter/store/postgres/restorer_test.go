// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
)

var _ app.KeyDumpLoader = (*postgres.Restorer)(nil)

func TestRestorerLoadsADumpIntoAnEmptyDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	keys := openKeys(t)
	want := seedKeys(t, keys)
	b := dump(t, keys)

	conn := testDatabase(t)
	r := postgres.NewRestorer(conn)
	if err := r.LoadDump(ctx, bytes.NewReader(b)); err != nil {
		t.Fatalf("LoadDump() error = %v", err)
	}
	restored, err := postgres.Open(ctx, conn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(restored.Close)
	got, err := restored.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("restored keys = %+v, want %+v", got, want)
	}
	if err := r.LoadDump(ctx, bytes.NewReader(b)); !errors.Is(err, postgres.ErrRestoreNotEmpty) {
		t.Errorf("second LoadDump() error = %v, want ErrRestoreNotEmpty", err)
	}
}
