// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

// OpenAuthenticator returns the authenticator of the API keys of the
// configured environment, stored in the configured database, and the
// function closing that database. The database is not reached until a key
// is checked.
func OpenAuthenticator(ctx context.Context, cfg config.Config) (*app.Authenticator, func(), error) {
	if err := cfg.RequireKeyStore(); err != nil {
		return nil, nil, err
	}
	env, err := domain.ParseKeyEnvironment(cfg.APIKeyEnv)
	if err != nil {
		return nil, nil, err
	}
	keys, err := Open(ctx, string(cfg.DatabaseURL.Bytes()))
	if err != nil {
		return nil, nil, err
	}
	return app.NewAuthenticator(keys, cfg.APIKeySecret.Bytes(), env), keys.Close, nil
}
