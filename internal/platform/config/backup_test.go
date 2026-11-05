// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

const (
	awsAccessKey = "AKIAXXXXXXXXXXXXXXXX"
	awsSecretKey = "aws-test-secret-key"
	recipient    = "age1pq1example"
)

// backupVars are a complete backup configuration.
func backupVars() map[string]string {
	return map[string]string{
		"ARNAKOFF_LOG_KEY":                  validKey,
		"ARNAKOFF_DATABASE_URL":             "postgres://backup@localhost/arnakoff",
		"ARNAKOFF_BACKUP_BUCKET":            "arnakoff-backups",
		"ARNAKOFF_BACKUP_BUCKET_ACCESS_KEY": awsAccessKey,
		"ARNAKOFF_BACKUP_BUCKET_SECRET_KEY": awsSecretKey,
		"ARNAKOFF_BACKUP_RECIPIENT":         recipient,
	}
}

func TestLoadReadsTheBackup(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(backupVars()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.RequireBackup(); err != nil {
		t.Fatalf("RequireBackup: %v", err)
	}
	b := cfg.Backup
	if b.Bucket != "arnakoff-backups" || b.Recipient != recipient ||
		string(b.AccessKey.Bytes()) != awsAccessKey || string(b.SecretKey.Bytes()) != awsSecretKey {
		t.Errorf("Backup = %+v, want arnakoff-backups with its keys and recipient", b)
	}
	if s := fmt.Sprintf("%v %+v", b, cfg); strings.Contains(s, awsAccessKey) || strings.Contains(s, awsSecretKey) {
		t.Errorf("formatted config shows a bucket key: %s", s)
	}
}

func TestRequireBackupNeedsEveryVariable(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"ARNAKOFF_DATABASE_URL", "ARNAKOFF_BACKUP_BUCKET", "ARNAKOFF_BACKUP_RECIPIENT"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			vars := backupVars()
			delete(vars, name)
			if name == "ARNAKOFF_BACKUP_BUCKET" {
				delete(vars, "ARNAKOFF_BACKUP_BUCKET_ACCESS_KEY")
				delete(vars, "ARNAKOFF_BACKUP_BUCKET_SECRET_KEY")
			}
			cfg, err := config.Load(env(vars))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if err := cfg.RequireBackup(); !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), name) {
				t.Errorf("RequireBackup() = %v, want ErrInvalid naming %s", err, name)
			}
		})
	}
}

func TestLoadRejectsAnInvalidBackupBucket(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		unset string
		set   map[string]string
	}{
		{name: "bucket without keys", unset: "ARNAKOFF_BACKUP_BUCKET_ACCESS_KEY"},
		{name: "keys without bucket", unset: "ARNAKOFF_BACKUP_BUCKET"},
		{name: "bad name", set: map[string]string{"ARNAKOFF_BACKUP_BUCKET": "Arnakoff/backups"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vars := backupVars()
			delete(vars, tt.unset)
			for k, v := range tt.set {
				vars[k] = v
			}
			_, err := config.Load(env(vars))
			if !errors.Is(err, config.ErrInvalid) {
				t.Fatalf("Load() error = %v, want ErrInvalid", err)
			}
			if strings.Contains(err.Error(), awsSecretKey) || strings.Contains(err.Error(), awsAccessKey) {
				t.Errorf("error %q quotes a bucket key", err)
			}
		})
	}
}
