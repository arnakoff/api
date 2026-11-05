// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/app"
)

// backupFunc is an httpapi.BackupRunner calling itself.
type backupFunc func(ctx context.Context) (string, error)

func (f backupFunc) Run(ctx context.Context) (string, error) { return f(ctx) }

func TestBackupHandler(t *testing.T) {
	t.Parallel()
	kept := backupFunc(func(context.Context) (string, error) { return "keys/x.tar.age", nil })
	tests := []struct {
		name   string
		backup backupFunc
		method string
		path   string
		want   int
		log    string
	}{
		{"alive", kept, http.MethodGet, "/healthz", http.StatusOK, ""},
		{"ready", kept, http.MethodGet, "/readyz", http.StatusOK, ""},
		{"kept", kept, http.MethodPost, "/backup", http.StatusNoContent, "keys/x.tar.age"},
		{"running", func(context.Context) (string, error) { return "", app.ErrBackupRunning }, http.MethodPost, "/backup", http.StatusConflict, "already running"},
		{"failed", func(context.Context) (string, error) { return "", errors.New("dump key store: refused") }, http.MethodPost, "/backup", http.StatusInternalServerError, "refused"},
		{"post only", kept, http.MethodGet, "/backup", http.StatusMethodNotAllowed, ""},
		{"nothing else", kept, http.MethodGet, "/v1/check", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			h := httpapi.NewBackupHandler(tt.backup, slog.New(slog.NewTextHandler(&logs, nil)))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, nil))
			if rec.Code != tt.want {
				t.Errorf("%s %s = %d, want %d", tt.method, tt.path, rec.Code, tt.want)
			}
			if !strings.Contains(logs.String(), tt.log) {
				t.Errorf("log %q, want it to mention %q", logs.String(), tt.log)
			}
		})
	}
}

func TestBackupHandlerBoundsTheRun(t *testing.T) {
	t.Parallel()
	var deadline bool
	h := httpapi.NewBackupHandler(backupFunc(func(ctx context.Context) (string, error) {
		_, deadline = ctx.Deadline()
		return "keys/x.tar.age", nil
	}), slog.New(slog.DiscardHandler))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/backup", nil))
	if !deadline {
		t.Error("the backup runs without a deadline")
	}
}
