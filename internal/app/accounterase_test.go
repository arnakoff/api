// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// eraserFake erases n, or fails with err, and keeps the address asked.
type eraserFake struct {
	n     app.PurgeCount
	err   error
	asked domain.Email
}

func (f *eraserFake) EraseAccount(_ context.Context, e domain.Email) (app.PurgeCount, error) {
	f.asked = e
	return f.n, f.err
}

func TestAccountEraseReturnsWhatTheStoreDeleted(t *testing.T) {
	t.Parallel()

	to := mustEmail(t, "dev@exemple.fr")
	store := &eraserFake{n: app.PurgeCount{Keys: 2, LoginRequests: 1}}
	got, err := app.NewAccountErase(store).Erase(context.Background(), to)
	if err != nil || got != store.n || store.asked != to {
		t.Errorf("Erase() = %+v, %v, asked %s; want %+v of the address", got, err, store.asked.Redacted(), store.n)
	}
}

func TestAccountEraseNamesTheAccountRedactedWhenTheStoreFails(t *testing.T) {
	t.Parallel()

	fail := errors.New("connection refused")
	_, err := app.NewAccountErase(&eraserFake{err: fail}).Erase(context.Background(), mustEmail(t, "dev@exemple.fr"))
	if !errors.Is(err, fail) {
		t.Fatalf("Erase() error = %v, want %v", err, fail)
	}
	if msg := err.Error(); strings.Contains(msg, "dev@") || !strings.Contains(msg, "***@exemple.fr") {
		t.Errorf("Erase() error = %q, want the address redacted", msg)
	}
}
