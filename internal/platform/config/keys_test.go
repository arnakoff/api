// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

// keySecret is 32 bytes in hex, other than validKey.
const keySecret = "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100"

// database is a connection URL; the password stands for its secret.
const database = "postgres://api:P4SSW0RD@db.example.eu/arnakoff?sslmode=verify-full" //nolint:gosec // G101: a fake password, to prove it is never printed.

func TestLoadReadsTheKeyStore(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_DATABASE_URL": database,
		"ARNAKOFF_API_KEY_SECRET": keySecret, "ARNAKOFF_API_KEY_ENV": "test",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(cfg.DatabaseURL.Bytes()) != database {
		t.Error("DatabaseURL is not the variable's value")
	}
	if len(cfg.APIKeySecret.Bytes()) != 32 {
		t.Errorf("APIKeySecret is %d bytes, want 32", len(cfg.APIKeySecret.Bytes()))
	}
	if cfg.APIKeyEnv != "test" {
		t.Errorf("APIKeyEnv = %q, want test", cfg.APIKeyEnv)
	}
	if err := cfg.RequireKeyStore(); err != nil {
		t.Errorf("RequireKeyStore: %v", err)
	}
}

func TestLoadLeavesTheKeyStoreOptional(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIKeyEnv != config.DefaultAPIKeyEnv {
		t.Errorf("APIKeyEnv = %q, want %q", cfg.APIKeyEnv, config.DefaultAPIKeyEnv)
	}
	err = cfg.RequireKeyStore()
	for _, name := range []string{"ARNAKOFF_DATABASE_URL", "ARNAKOFF_API_KEY_SECRET"} {
		if !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
			t.Errorf("RequireKeyStore() = %v, want ErrInvalid naming %s", err, name)
		}
	}
}

func TestLoadRejectsInvalidKeyStoreValues(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]string{
		"ARNAKOFF_API_KEY_SECRET": "0011",
		"ARNAKOFF_API_KEY_ENV":    "prod",
	} {
		_, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey, name: value}))
		if !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
			t.Errorf("Load(%s=%q) = %v, want ErrInvalid naming it", name, value, err)
		}
	}
}

func TestRequireKeyStoreRejectsTheLogKey(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_DATABASE_URL": database, "ARNAKOFF_API_KEY_SECRET": validKey,
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.RequireKeyStore(); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("RequireKeyStore() with the log key as API key secret = %v, want ErrInvalid", err)
	}
}

func TestKeyStoreSecretsNeverPrinted(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_DATABASE_URL": database, "ARNAKOFF_API_KEY_SECRET": keySecret,
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s := strings.ToLower(fmt.Sprintf("%+v", cfg)); strings.Contains(s, "p4ssw0rd") || strings.Contains(s, keySecret) {
		t.Errorf("the configuration prints a secret: %s", s)
	}
}

func TestRequireDatabaseIgnoresTheSecret(t *testing.T) {
	t.Parallel()

	if err := (config.Config{DatabaseURL: config.Secret(database)}).RequireDatabase(); err != nil {
		t.Errorf("RequireDatabase() with a database = %v, want nil", err)
	}
	err := config.Config{}.RequireDatabase()
	if !errors.Is(err, config.ErrInvalid) || strings.Contains(err.Error(), "ARNAKOFF_API_KEY_SECRET") {
		t.Errorf("RequireDatabase() = %v, want ErrInvalid naming the database only", err)
	}
}

func TestLoadReadsTheDatabaseCA(t *testing.T) {
	t.Parallel()
	ca := "-----BEGIN CERTIFICATE-----\nAQID\n-----END CERTIFICATE-----\n"
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_DATABASE_CA": ca}))
	if err != nil || cfg.DatabaseCA != ca {
		t.Fatalf("Load() = %q, %v; want the PEM as given", cfg.DatabaseCA, err)
	}
	for name, value := range map[string]string{
		"not PEM":     "a certificate",
		"no CERT":     "-----BEGIN PRIVATE KEY-----\nAQID\n-----END PRIVATE KEY-----\n",
		"stray bytes": "junk",
	} {
		_, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_DATABASE_CA": value}))
		if !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), "ARNAKOFF_DATABASE_CA") {
			t.Errorf("%s: error = %v, want ErrInvalid naming ARNAKOFF_DATABASE_CA", name, err)
		}
	}
}
