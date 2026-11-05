// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/app"
)

func TestOnceIsReadyOnceRun(t *testing.T) {
	t.Parallel()

	calls := 0
	o := app.NewOnce(func(context.Context) error { calls++; return nil })
	if o.Ready() {
		t.Fatal("Ready() before Run = true, want false")
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !o.Ready() || calls != 1 {
		t.Errorf("after Run: Ready() = %v, step run %d times; want true, 1", o.Ready(), calls)
	}
}

func TestOnceIsNotReadyWhenItFails(t *testing.T) {
	t.Parallel()

	failed := errors.New("connection refused")
	o := app.NewOnce(func(context.Context) error { return failed })
	if err := o.Run(context.Background()); !errors.Is(err, failed) {
		t.Fatalf("Run() error = %v, want %v", err, failed)
	}
	if o.Ready() {
		t.Error("Ready() after a failed Run = true, want false")
	}
}
