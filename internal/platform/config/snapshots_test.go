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
	bucketAccessKey = "SCWXXXXXXXXXXXXXXXXX"
	bucketSecretKey = "11111111-2222-3333-4444-555555555555"
)

func TestLoadKeepsTheSnapshotsOnDiskByDefault(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SnapshotBucket.Enabled() {
		t.Errorf("SnapshotBucket = %+v, want off", cfg.SnapshotBucket)
	}
}

func TestLoadReadsTheSnapshotBucket(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY":                    validKey,
		"ARNAKOFF_SNAPSHOT_BUCKET":            "arnakoff-snapshots",
		"ARNAKOFF_SNAPSHOT_BUCKET_ACCESS_KEY": bucketAccessKey,
		"ARNAKOFF_SNAPSHOT_BUCKET_SECRET_KEY": bucketSecretKey,
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	b := cfg.SnapshotBucket
	if !b.Enabled() || b.Name != "arnakoff-snapshots" ||
		string(b.AccessKey.Bytes()) != bucketAccessKey || string(b.SecretKey.Bytes()) != bucketSecretKey {
		t.Errorf("SnapshotBucket = %+v, want arnakoff-snapshots with its keys", b)
	}
	if s := fmt.Sprintf("%v %+v", b, cfg); strings.Contains(s, bucketAccessKey) || strings.Contains(s, bucketSecretKey) {
		t.Errorf("formatted config shows a bucket key: %s", s)
	}
}

func TestLoadRejectsAnInvalidSnapshotBucket(t *testing.T) {
	t.Parallel()
	keys := map[string]string{
		"ARNAKOFF_SNAPSHOT_BUCKET_ACCESS_KEY": bucketAccessKey,
		"ARNAKOFF_SNAPSHOT_BUCKET_SECRET_KEY": bucketSecretKey,
	}
	for _, tt := range []struct {
		name string
		vars map[string]string
	}{
		{"bucket without keys", map[string]string{"ARNAKOFF_SNAPSHOT_BUCKET": "arnakoff-snapshots"}},
		{"keys without bucket", keys},
		{"secret key missing", map[string]string{
			"ARNAKOFF_SNAPSHOT_BUCKET": "arnakoff-snapshots", "ARNAKOFF_SNAPSHOT_BUCKET_ACCESS_KEY": bucketAccessKey,
		}},
		{"upper case", withBucket(keys, "Arnakoff")},
		{"too short", withBucket(keys, "ab")},
		{"too long", withBucket(keys, strings.Repeat("a", 64))},
		{"slash", withBucket(keys, "arnakoff/snapshots")},
		{"dot segment", withBucket(keys, "..")},
		{"ends with a dash", withBucket(keys, "arnakoff-")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.vars["ARNAKOFF_LOG_KEY"] = validKey
			_, err := config.Load(env(tt.vars))
			if !errors.Is(err, config.ErrInvalid) {
				t.Fatalf("Load() error = %v, want ErrInvalid", err)
			}
			if strings.Contains(err.Error(), bucketSecretKey) || strings.Contains(err.Error(), bucketAccessKey) {
				t.Errorf("error %q quotes a bucket key", err)
			}
		})
	}
}

// withBucket returns a copy of keys with the bucket name set to name.
func withBucket(keys map[string]string, name string) map[string]string {
	vars := map[string]string{"ARNAKOFF_SNAPSHOT_BUCKET": name}
	for k, v := range keys {
		vars[k] = v
	}
	return vars
}
