// SPDX-License-Identifier: AGPL-3.0-only

package assembly_test

import (
	"errors"
	"testing"

	"filippo.io/age"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/backupseal"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/platform/config"
)

const presignedDump = "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com/restore/keys.tar.age?X-Amz-Signature=0123abcd"

// restoreConfig is a complete restore configuration. Nothing connects
// until the restore runs.
func restoreConfig(url, identity string) config.Config {
	return config.Config{
		DatabaseURL: config.Secret("postgres://admin@localhost/arnakoff"),
		Restore:     config.Restore{URL: config.Secret(url), Identity: config.Secret(identity)},
	}
}

func TestKeyRestoreIsBuiltFromACompleteConfig(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	r, err := assembly.KeyRestore(restoreConfig(presignedDump, id.String()))
	if err != nil {
		t.Fatalf("KeyRestore() error = %v", err)
	}
	if r == nil {
		t.Error("KeyRestore() = nil")
	}
}

func TestKeyRestoreRefuses(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		cfg  config.Config
		want error
	}{
		"unconfigured":   {config.Config{}, config.ErrInvalid},
		"a bad URL":      {restoreConfig("https://example.org/keys.tar.age?X-Amz-Signature=0123abcd", id.String()), objectstore.ErrPresignedURL},
		"a bad identity": {restoreConfig(presignedDump, id.Recipient().String()), backupseal.ErrIdentity},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := assembly.KeyRestore(tt.cfg); !errors.Is(err, tt.want) {
				t.Errorf("KeyRestore() error = %v, want %v", err, tt.want)
			}
		})
	}
}
