// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

const eraseEmail = "titulaire@exemple.fr"

// eraseVars are a complete erasure configuration.
func eraseVars() map[string]string {
	return map[string]string{
		"ARNAKOFF_LOG_KEY":      validKey,
		"ARNAKOFF_DATABASE_URL": "postgres://admin@localhost/arnakoff",
		"ARNAKOFF_ERASE_EMAIL":  eraseEmail,
	}
}

func TestLoadReadsTheAddressToErase(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(eraseVars()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.RequireErase(); err != nil {
		t.Fatalf("RequireErase: %v", err)
	}
	if got := string(cfg.Erase.Email.Bytes()); got != eraseEmail {
		t.Errorf("EraseEmail = %q, want the address", got)
	}
	if s := fmt.Sprintf("%v %+v", cfg.Erase.Email, cfg); strings.Contains(s, "titulaire") {
		t.Errorf("formatted config shows the address: %s", s)
	}
}

func TestRequireEraseNeedsEveryVariable(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"ARNAKOFF_DATABASE_URL", "ARNAKOFF_ERASE_EMAIL"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			vars := eraseVars()
			delete(vars, name)
			cfg, err := config.Load(env(vars))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if err := cfg.RequireErase(); !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
				t.Errorf("RequireErase() = %v, want ErrInvalid naming %s", err, name)
			}
		})
	}
}
