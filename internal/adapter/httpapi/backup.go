// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/arnakoff/api/internal/app"
)

// backupPath is the route the daily trigger of the backup container posts
// to (#75).
const backupPath = "/backup"

// backupTimeout bounds a backup, under the server's writeTimeout.
const backupTimeout = writeTimeout - 5*time.Second

// BackupRunner runs one backup and returns the name it is kept under.
type BackupRunner interface {
	Run(ctx context.Context) (string, error)
}

// alwaysReady is the readiness of a process ready once it serves.
type alwaysReady struct{}

func (alwaysReady) Ready() bool { return true }

// NewBackupHandler serves the probes and POST /backup, which runs b: 204
// once the backup is kept, 409 while another one runs, 500 when it fails.
// It is the handler of `admin backup -listen`, inside the private network,
// called by a cron trigger only.
func NewBackupHandler(b BackupRunner, logger *slog.Logger) http.Handler {
	mux := probeMux(alwaysReady{})
	mux.Handle("POST "+backupPath, runBackup(b, logger))
	return mux
}

// runBackup runs b for each request, within backupTimeout.
func runBackup(b BackupRunner, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), backupTimeout)
		defer cancel()
		name, err := b.Run(ctx)
		switch {
		case errors.Is(err, app.ErrBackupRunning):
			logger.WarnContext(ctx, "backup not started", "error", err.Error())
			writeProbe(w, http.StatusConflict, `{"status":"en cours"}`)
		case err != nil:
			logger.ErrorContext(ctx, "backup failed", "error", err.Error())
			writeProbe(w, http.StatusInternalServerError, `{"status":"échec"}`)
		default:
			logger.InfoContext(ctx, "backup kept", "object", name)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
