// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
)

func TestOpenKeyServicesRequiresTheKeyStore(t *testing.T) {
	t.Parallel()

	if _, err := postgres.OpenKeyServices(t.Context(), config.Config{APIKeyEnv: "live"}); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("OpenKeyServices() without a database error = %v, want ErrInvalid", err)
	}
}

// The database is not reached until a key is checked.
func TestOpenKeyServicesOpensLazily(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DatabaseURL:  config.Secret("postgres://api@127.0.0.1:1/arnakoff"),
		APIKeySecret: config.Secret("a test secret of thirty-two byte"),
		APIKeyEnv:    "test",
	}
	svc, err := postgres.OpenKeyServices(t.Context(), cfg)
	if err != nil {
		t.Fatalf("OpenKeyServices() error = %v", err)
	}
	defer svc.Close()
	if svc.Auth == nil || svc.Purge == nil || svc.WatchRevocations == nil {
		t.Errorf("OpenKeyServices() = %+v, want an authenticator, a purge and a watch", svc)
	}
}

// While the watch of revocations runs, a revoked key is refused within
// moments, though valid keys are kept in memory.
func TestOpenKeyServicesRefusesARevokedKeyAtOnce(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	conn := testDatabase(t)
	if err := postgres.Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	cfg := config.Config{
		DatabaseURL:  config.Secret(conn),
		APIKeySecret: config.Secret("a test secret of thirty-two byte"),
		APIKeyEnv:    "test",
	}
	keys, err := postgres.Open(ctx, conn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(keys.Close)
	admin := app.NewKeyAdmin(keys, cfg.APIKeySecret.Bytes(), clock.System{}, rand.Reader)
	key, rec, err := admin.Issue(ctx, domain.KeyTest, "bot")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	svc, err := postgres.OpenKeyServices(ctx, cfg)
	if err != nil {
		t.Fatalf("OpenKeyServices() error = %v", err)
	}
	defer svc.Close()
	watching, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); svc.WatchRevocations(watching, func(error) {}) }()
	t.Cleanup(func() { stop(); <-done })

	if _, err := svc.Auth.Authenticate(ctx, key.String()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if err := admin.Revoke(ctx, rec.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err := svc.Auth.Authenticate(ctx, key.String())
		if errors.Is(err, app.ErrKeyRevoked) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Authenticate() 5 s after the revocation error = %v, want ErrKeyRevoked", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
