// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// EraseAccount deletes, in one transaction, every key of the account of
// e, active or revoked, and every login request to it (those stored before
// migration 0004 excepted: they have no account), and returns how
// many of each it deleted. It holds the lock of the account, so that no
// key of it is stored meanwhile by ReplaceAccountKey. Each deleted key is
// announced on the channel of revocations (migration 0003). It implements
// app.AccountEraser.
func (k *Keys) EraseAccount(ctx context.Context, e domain.Email) (app.PurgeCount, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	account := e.Account()
	var n app.PurgeCount
	err := pgx.BeginFunc(ctx, k.pool, func(tx pgx.Tx) error {
		if err := lockAccount(ctx, tx, account); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, "DELETE FROM api_keys WHERE account = $1", account)
		if err != nil {
			return fmt.Errorf("delete account keys: %w", err)
		}
		n.Keys = tag.RowsAffected()
		if tag, err = tx.Exec(ctx, "DELETE FROM login_requests WHERE account = $1", account); err != nil {
			return fmt.Errorf("delete account login requests: %w", err)
		}
		n.LoginRequests = tag.RowsAffected()
		return nil
	})
	if err != nil {
		return app.PurgeCount{}, fmt.Errorf("erase account: %w", err)
	}
	return n, nil
}

// lockAccount takes, until the end of tx, the lock that serialises the
// changes to the keys of account.
func lockAccount(ctx context.Context, tx pgx.Tx, account string) error {
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", account); err != nil {
		return fmt.Errorf("lock account: %w", err)
	}
	return nil
}
