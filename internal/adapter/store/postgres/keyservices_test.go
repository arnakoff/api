// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"crypto/rand"
	"errors"
	"testing"

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
	if svc.Auth == nil || svc.Purge == nil {
		t.Errorf("OpenKeyServices() = %+v, want an authenticator and a purge", svc)
	}
}

// A valid key is checked against the database once, then kept for
// app.KeyCacheTTL: its revocation is not seen until then.
func TestOpenKeyServicesCachesValidKeys(t *testing.T) {
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
	auth := svc.Auth
	if _, err := auth.Authenticate(ctx, key.String()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if err := admin.Revoke(ctx, rec.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, err := auth.Authenticate(ctx, key.String()); err != nil {
		t.Errorf("Authenticate() right after the revocation error = %v, want the cached key accepted", err)
	}
}
