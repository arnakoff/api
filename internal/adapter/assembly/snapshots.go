// SPDX-License-Identifier: AGPL-3.0-only

package assembly

import (
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
)

// SnapshotStore returns where the snapshots of cfg are stored: its bucket
// when it is enabled, in production (ADR 0005), else the directory
// cfg.SnapshotDir, created if missing. The sync and API processes share it.
func SnapshotStore(cfg config.Config) (app.SnapshotStore, error) {
	b := cfg.SnapshotBucket
	if !b.Enabled() {
		return store.NewDisk(cfg.SnapshotDir())
	}
	client, err := httpclient.New(httpclient.Config{AllowedHosts: []string{objectstore.Host}})
	if err != nil {
		return nil, err
	}
	return objectstore.New(objectstore.Options{
		Bucket:    b.Name,
		AccessKey: string(b.AccessKey.Bytes()),
		SecretKey: b.SecretKey,
		HTTP:      client,
	})
}
