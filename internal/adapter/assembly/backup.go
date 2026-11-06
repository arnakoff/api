// SPDX-License-Identifier: AGPL-3.0-only

package assembly

import (
	"context"

	"github.com/arnakoff/api/internal/adapter/backupseal"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
)

// Backup returns the backup of the key store of cfg, sealed for its
// recipient and kept in its bucket, and the function that closes its
// database pool. Nothing connects until a backup runs.
func Backup(ctx context.Context, cfg config.Config) (*app.Backup, func(), error) {
	if err := cfg.RequireBackup(); err != nil {
		return nil, nil, err
	}
	b := cfg.Backup
	signer, err := backupseal.NewSigner(string(b.SigningKey.Bytes()))
	if err != nil {
		return nil, nil, err
	}
	sealer, err := backupseal.New(b.Recipient, signer)
	if err != nil {
		return nil, nil, err
	}
	client, err := httpclient.New(httpclient.Config{AllowedHosts: []string{objectstore.AWSParisHost}})
	if err != nil {
		return nil, nil, err
	}
	archive, err := objectstore.NewArchive(objectstore.Options{
		Endpoint:  "https://" + objectstore.AWSParisHost,
		Region:    objectstore.AWSParisRegion,
		Bucket:    b.Bucket,
		AccessKey: string(b.AccessKey.Bytes()),
		SecretKey: b.SecretKey,
		HTTP:      client,
	})
	if err != nil {
		return nil, nil, err
	}
	keys, err := postgres.Open(ctx, string(cfg.DatabaseURL.Bytes()), postgres.WithCA([]byte(cfg.DatabaseCA)))
	if err != nil {
		return nil, nil, err
	}
	return app.NewBackup(keys, sealer, archive, clock.System{}), keys.Close, nil
}
