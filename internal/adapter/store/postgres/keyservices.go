// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
)

// KeyServices are the use cases of the API process on the API keys of the
// configured database, sharing one pool of connections.
type KeyServices struct {
	// Auth accepts the active keys of the configured environment; a valid
	// key is kept in memory for app.KeyCacheTTL while WatchRevocations
	// listens, and not at all otherwise.
	Auth *app.Authenticator
	// WatchRevocations makes Auth forget each revoked key at once, until
	// ctx is done; it reports each loss of its connection to report.
	WatchRevocations func(ctx context.Context, report func(error))
	// Purge deletes the keys revoked longer than
	// domain.RevokedKeyRetention ago, every app.KeyPurgeEvery.
	Purge *app.KeyPurge
	// Close closes the database, waiting for the connections in use.
	Close func()
	// Store is the database itself: the login requests and the account
	// keys of app.KeyRequests.
	Store *Keys
	// Environment is that of the keys accepted and issued.
	Environment domain.KeyEnvironment
}

// watchRetry is how long the watch of revocations waits before it
// connects again.
const watchRetry = 5 * time.Second

// OpenKeyServices returns the KeyServices of the configured database. The
// database is not reached until a key is checked or purged.
func OpenKeyServices(ctx context.Context, cfg config.Config) (KeyServices, error) {
	if err := cfg.RequireKeyStore(); err != nil {
		return KeyServices{}, err
	}
	env, err := domain.ParseKeyEnvironment(cfg.APIKeyEnv)
	if err != nil {
		return KeyServices{}, err
	}
	keys, err := Open(ctx, string(cfg.DatabaseURL.Bytes()), WithCA([]byte(cfg.DatabaseCA)))
	if err != nil {
		return KeyServices{}, err
	}
	cached := app.NewKeyCache(keys, clock.System{}, app.KeyCacheTTL)
	return KeyServices{
		Auth: app.NewAuthenticator(cached, cfg.APIKeySecret.Bytes(), env),
		WatchRevocations: func(ctx context.Context, report func(error)) {
			keys.WatchRevocations(ctx, cached, watchRetry, report)
		},
		Purge: app.NewKeyPurge(keys, clock.System{}, clock.System{}, app.KeyPurgeEvery),
		Close: keys.Close,
		Store: keys, Environment: env,
	}, nil
}
