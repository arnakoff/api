// SPDX-License-Identifier: AGPL-3.0-only

package assembly_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/backupseal"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/platform/config"
)

// A throwaway test pair (backupseal tests): seed and public key.
const (
	testSigningKey = "233b01916151fe156d809150c20bb942c8692d2b76eec18ba46e77863ac2c3e5"
	testVerifyKey  = "07b769cae872f43683b42b114b802a78e5c1cd1b5a58a6e3cf4e9868605e4873"
)

const presignedDump = "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com/restore/keys.tar.age?X-Amz-Signature=0123abcd"

// restoreConfig is a complete restore configuration. Nothing connects
// until the restore runs.
func restoreConfig(url, identity string) config.Config {
	return config.Config{
		DatabaseURL: config.Secret("postgres://admin@localhost/arnakoff"),
		Restore:     config.Restore{URL: config.Secret(url), Identity: config.Secret(identity), VerifyKey: testVerifyKey},
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
		"a bad verify key": {func() config.Config {
			c := restoreConfig(presignedDump, id.String())
			c.Restore.VerifyKey = "zz"
			return c
		}(), backupseal.ErrVerifyKey},
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

func TestFileRestoreRefusesAnUnsignedDumpBeforeTheDatabase(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "keys.tar")
	if err := os.WriteFile(file, []byte("a clear dump nobody signed, long enough to hold a signature trailer..........................."), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		DatabaseURL: config.Secret("postgres://admin@localhost:1/arnakoff"),
		Restore:     config.Restore{VerifyKey: testVerifyKey},
	}
	r, err := assembly.FileRestore(cfg, file)
	if err != nil {
		t.Fatalf("FileRestore() error = %v", err)
	}
	if err := r.Run(t.Context()); !errors.Is(err, backupseal.ErrSignature) {
		t.Errorf("Run() error = %v, want ErrSignature before any connection", err)
	}
	if _, err := assembly.FileRestore(config.Config{}, file); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("FileRestore() unconfigured error = %v, want ErrInvalid", err)
	}
}
