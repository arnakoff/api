// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/platform/config"
)

func TestOpenAuthenticatorRequiresTheKeyStore(t *testing.T) {
	t.Parallel()

	if _, _, err := postgres.OpenAuthenticator(t.Context(), config.Config{APIKeyEnv: "live"}); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("OpenAuthenticator() without a database error = %v, want ErrInvalid", err)
	}
}

// The database is not reached until a key is checked.
func TestOpenAuthenticatorOpensLazily(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DatabaseURL:  config.Secret("postgres://api@127.0.0.1:1/arnakoff"),
		APIKeySecret: config.Secret("a test secret of thirty-two byte"),
		APIKeyEnv:    "test",
	}
	auth, closeKeys, err := postgres.OpenAuthenticator(t.Context(), cfg)
	if err != nil {
		t.Fatalf("OpenAuthenticator() error = %v", err)
	}
	defer closeKeys()
	if auth == nil {
		t.Error("OpenAuthenticator() = nil")
	}
}
