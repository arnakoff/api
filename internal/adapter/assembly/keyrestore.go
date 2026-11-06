// SPDX-License-Identifier: AGPL-3.0-only

package assembly

import (
	"context"
	"os"

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
	verifier, err := backupseal.NewVerifier(cfg.Restore.VerifyKey)
	if err != nil {
		return nil, err
	}
	opener, err := backupseal.NewOpener(string(cfg.Restore.Identity.Bytes()), verifier)
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

// FileRestore returns the restore of the clear dump in file, which the
// owner decrypted, into the configured database. Its signature is checked
// with the configured verify key before anything connects.
func FileRestore(cfg config.Config, file string) (*app.KeyRestore, error) {
	if err := cfg.RequireFileRestore(); err != nil {
		return nil, err
	}
	verifier, err := backupseal.NewVerifier(cfg.Restore.VerifyKey)
	if err != nil {
		return nil, err
	}
	loader := postgres.NewRestorer(string(cfg.DatabaseURL.Bytes()), postgres.WithCA([]byte(cfg.DatabaseCA)))
	return app.NewKeyRestore(clearDumpFile(file), verifier, loader), nil
}

// clearDumpFile reads the clear dump the operator names. It implements
// app.KeyDumpFetcher.
type clearDumpFile string

// FetchDump returns the content of the file.
func (f clearDumpFile) FetchDump(context.Context) ([]byte, error) {
	return os.ReadFile(string(f))
}
