// SPDX-License-Identifier: AGPL-3.0-only

package assembly_test

import (
	"errors"
	"testing"

	"filippo.io/age"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/backupseal"
	"github.com/arnakoff/api/internal/platform/config"
)

// backupConfig is a complete backup configuration for recipient. Nothing
// connects until a backup runs.
func backupConfig(recipient string) config.Config {
	return config.Config{
		DatabaseURL: config.Secret("postgres://backup@localhost/arnakoff"),
		Backup: config.Backup{
			Bucket: "arnakoff-backups", AccessKey: config.Secret("AKIAXXXXXXXXXXXXXXXX"), SecretKey: config.Secret("s3cr3t"),
			Recipient: recipient,
		},
	}
}

func TestBackupIsBuiltFromACompleteConfig(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	b, closeBackup, err := assembly.Backup(t.Context(), backupConfig(id.Recipient().String()))
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	t.Cleanup(closeBackup)
	if b == nil {
		t.Error("Backup() = nil")
	}
}

func TestBackupRefuses(t *testing.T) {
	t.Parallel()
	if _, _, err := assembly.Backup(t.Context(), backupConfig("not a key")); !errors.Is(err, backupseal.ErrRecipient) {
		t.Errorf("Backup() with a bad recipient error = %v, want ErrRecipient", err)
	}
	if _, _, err := assembly.Backup(t.Context(), config.Config{}); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("Backup() unconfigured error = %v, want ErrInvalid", err)
	}
}
