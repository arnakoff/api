// SPDX-License-Identifier: AGPL-3.0-only

package assembly

import (
	"github.com/arnakoff/api/internal/adapter/backupseal"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
)

// KeyRestore returns the restore of the key store of cfg: the dump at its
// presigned URL, fetched by a client that may reach that host only, opened
// with its identity and loaded into its database. Nothing connects until
// the restore runs.
func KeyRestore(cfg config.Config) (*app.KeyRestore, error) {
	if err := cfg.RequireRestore(); err != nil {
		return nil, err
	}
	u, err := objectstore.ParsePresigned(string(cfg.Restore.URL.Bytes()))
	if err != nil {
		return nil, err
	}
	opener, err := backupseal.NewOpener(string(cfg.Restore.Identity.Bytes()))
	if err != nil {
		return nil, err
	}
	client, err := httpclient.New(httpclient.Config{AllowedHosts: []string{u.Host()}})
	if err != nil {
		return nil, err
	}
	loader := postgres.NewRestorer(string(cfg.DatabaseURL.Bytes()), postgres.WithCA([]byte(cfg.DatabaseCA)))
	return app.NewKeyRestore(objectstore.NewPresignedObject(u, client), opener, loader), nil
}
