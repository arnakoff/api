// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
)

// startMigration serves migrate on a loopback listener until the end of t
// and returns the URL of its readiness probe.
func startMigration(t *testing.T, migrate app.MigrateFunc) string {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveMigration(ctx, ln, app.NewMigration(migrate), slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("serveMigration() after stop = %v, want nil", err)
		}
	})
	return "http://" + ln.Addr().String() + "/readyz"
}

// readyStatus returns the status of GET url.
func readyStatus(t *testing.T, url string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestServeMigrationIsReadyOnceApplied(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	url := startMigration(t, func(context.Context) error { <-release; return nil })
	if got := readyStatus(t, url); got != http.StatusServiceUnavailable {
		t.Fatalf("/readyz while migrating = %d, want 503", got)
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for readyStatus(t, url) != http.StatusOK {
		if time.Now().After(deadline) {
			t.Fatal("/readyz not 200 within 5 s of the migrations")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A failed migration keeps the probes served, /readyz at 503, so that the
// platform shows the container unready and its logs stay readable.
func TestServeMigrationStaysUnreadyWhenItFails(t *testing.T) {
	t.Parallel()

	ran := make(chan struct{})
	url := startMigration(t, func(context.Context) error { close(ran); return errors.New("connection refused") })
	<-ran
	if got := readyStatus(t, url); got != http.StatusServiceUnavailable {
		t.Errorf("/readyz after a failed migration = %d, want 503", got)
	}
}
