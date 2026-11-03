// SPDX-License-Identifier: AGPL-3.0-only

package assembly_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/platform/config"
)

func TestSnapshotStoreIsTheDiskByDefault(t *testing.T) {
	t.Parallel()
	cfg := config.Config{DataDir: t.TempDir()}

	st, err := assembly.SnapshotStore(cfg)
	if err != nil {
		t.Fatalf("SnapshotStore() error = %v", err)
	}
	if _, ok := st.(*store.Disk); !ok {
		t.Errorf("SnapshotStore() = %T, want *store.Disk", st)
	}
}

func TestSnapshotStoreIsTheBucketWhenConfigured(t *testing.T) {
	t.Parallel()
	dataDir := filepath.Join(t.TempDir(), "unused")
	cfg := config.Config{DataDir: dataDir, SnapshotBucket: config.SnapshotBucket{
		Name: "arnakoff-snapshots", AccessKey: config.Secret("SCWXXXXXXXXXXXXXXXXX"), SecretKey: config.Secret("s3cr3t"),
	}}

	st, err := assembly.SnapshotStore(cfg)
	if err != nil {
		t.Fatalf("SnapshotStore() error = %v", err)
	}
	if _, ok := st.(*objectstore.Store); !ok {
		t.Errorf("SnapshotStore() = %T, want *objectstore.Store", st)
	}
	if _, err := os.Stat(dataDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stat %s error = %v, want no snapshot directory created", dataDir, err)
	}
}
