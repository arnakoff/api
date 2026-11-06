// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

const (
	restoreURL      = "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com/restore/keys.tar.age?X-Amz-Signature=0123abcd"
	restoreIdentity = "AGE-SECRET-KEY-PQ-1EXAMPLE"
	verifyKey       = "07b769cae872f43683b42b114b802a78e5c1cd1b5a58a6e3cf4e9868605e4873"
)

// restoreVars are a complete restore configuration.
func restoreVars() map[string]string {
	return map[string]string{
		"ARNAKOFF_LOG_KEY":            validKey,
		"ARNAKOFF_DATABASE_URL":       "postgres://admin@localhost/arnakoff",
		"ARNAKOFF_RESTORE_URL":        restoreURL,
		"ARNAKOFF_RESTORE_IDENTITY":   restoreIdentity,
		"ARNAKOFF_RESTORE_VERIFY_KEY": verifyKey,
	}
}

func TestLoadReadsTheRestore(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(restoreVars()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.RequireRestore(); err != nil {
		t.Fatalf("RequireRestore: %v", err)
	}
	r := cfg.Restore
	if string(r.URL.Bytes()) != restoreURL || string(r.Identity.Bytes()) != restoreIdentity || r.VerifyKey != verifyKey {
		t.Errorf("Restore = %+v, want its URL and identity", r)
	}
	if s := fmt.Sprintf("%v %+v", r, cfg); strings.Contains(s, "0123abcd") || strings.Contains(s, restoreIdentity) {
		t.Errorf("formatted config shows a restore secret: %s", s)
	}
}

func TestRequireRestoreNeedsEveryVariable(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"ARNAKOFF_DATABASE_URL", "ARNAKOFF_RESTORE_URL", "ARNAKOFF_RESTORE_IDENTITY", "ARNAKOFF_RESTORE_VERIFY_KEY"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			vars := restoreVars()
			delete(vars, name)
			cfg, err := config.Load(env(vars))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if err := cfg.RequireRestore(); !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
				t.Errorf("RequireRestore() = %v, want ErrInvalid naming %s", err, name)
			}
		})
	}
}

func TestRequireFileRestoreNeedsTheDatabaseAndTheVerifyKey(t *testing.T) {
	t.Parallel()
	vars := map[string]string{
		"ARNAKOFF_LOG_KEY":            validKey,
		"ARNAKOFF_DATABASE_URL":       "postgres://admin@localhost/arnakoff",
		"ARNAKOFF_RESTORE_VERIFY_KEY": verifyKey,
	}
	cfg, err := config.Load(env(vars))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.RequireFileRestore(); err != nil {
		t.Errorf("RequireFileRestore() = %v, want nil: a clear dump needs no URL nor identity", err)
	}
	for _, name := range []string{"ARNAKOFF_DATABASE_URL", "ARNAKOFF_RESTORE_VERIFY_KEY"} {
		v := maps.Clone(vars)
		delete(v, name)
		cfg, err := config.Load(env(v))
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if err := cfg.RequireFileRestore(); !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
			t.Errorf("RequireFileRestore() without %s = %v, want ErrInvalid naming it", name, err)
		}
	}
}
