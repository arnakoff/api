// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
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

func TestEraseStepErasesTheAddressAndLogsWhatItDeleted(t *testing.T) {
	t.Parallel()

	store := &eraserFake{n: app.PurgeCount{Keys: 2, LoginRequests: 1}}
	var logs bytes.Buffer
	step, err := eraseStep(app.NewAccountErase(store), "Dev@Exemple.fr", slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("eraseStep() error = %v", err)
	}
	if err := step(t.Context()); err != nil {
		t.Fatalf("step() error = %v", err)
	}
	if store.asked.Account() != "dev@exemple.fr" {
		t.Errorf("erased %s, want the configured address", store.asked.Redacted())
	}
	if got := logs.String(); !strings.Contains(got, "keys=2") || !strings.Contains(got, "login_requests=1") {
		t.Errorf("log = %q, want the counts", got)
	}
}

func TestEraseStepRefusesAnAddressItCannotParse(t *testing.T) {
	t.Parallel()

	_, err := eraseStep(app.NewAccountErase(&eraserFake{}), "not an address", slog.Default())
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Errorf("eraseStep() error = %v, want ErrInvalidEmail", err)
	}
}
