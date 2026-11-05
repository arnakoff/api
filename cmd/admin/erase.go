// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"log/slog"
	"net"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

// runErase erases the account of ARNAKOFF_ERASE_EMAIL, then exits, or
// serves its probes on listen when it is set.
func runErase(ctx context.Context, cfg config.Config, listen string) error {
	if err := cfg.RequireErase(); err != nil {
		return err
	}
	keys, err := postgres.Open(ctx, string(cfg.DatabaseURL.Bytes()), postgres.WithCA([]byte(cfg.DatabaseCA)))
	if err != nil {
		return err
	}
	defer keys.Close()
	step, err := eraseStep(app.NewAccountErase(keys), string(cfg.Erase.Email.Bytes()), slog.Default())
	if err != nil {
		return err
	}
	if listen == "" {
		return step(ctx)
	}
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", listen)
	if err != nil {
		return err
	}
	return serveOnce(ctx, ln, "erase", app.NewOnce(step), slog.Default())
}

// eraseStep returns the erasure of the account of the address raw, which
// logs what it deleted to logger. The address is logged as a
// domain.Email, which the redacting logger turns into its pseudonym.
func eraseStep(erase *app.AccountErase, raw string, logger *slog.Logger) (func(context.Context) error, error) {
	e, err := domain.ParseEmail(raw)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		n, err := erase.Erase(ctx, e)
		if err != nil {
			return err
		}
		logger.InfoContext(ctx, "account erased", "account", e, "keys", n.Keys, "login_requests", n.LoginRequests)
		return nil
	}, nil
}
