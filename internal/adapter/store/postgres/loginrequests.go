// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// CreateLoginRequest stores r in the login_requests table. It implements
// app.LoginRequestStore.
func (k *Keys) CreateLoginRequest(ctx context.Context, r domain.LoginRequest) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	_, err := k.pool.Exec(ctx,
		`INSERT INTO login_requests (token_digest, email, created_at, expires_at) VALUES ($1, $2, $3, $4)`,
		r.Digest[:], r.Email.String(), r.CreatedAt, r.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insert login request: %w", err)
	}
	return nil
}

// UseLoginRequest marks used at at the request of digest d, unused and
// unexpired at at, and returns its address, or app.ErrLoginLinkInvalid.
// One statement checks and marks, so two calls never both succeed.
func (k *Keys) UseLoginRequest(ctx context.Context, d domain.TokenDigest, at time.Time) (domain.Email, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	var raw string
	err := k.pool.QueryRow(ctx,
		`UPDATE login_requests SET used_at = $2
		 WHERE token_digest = $1 AND used_at IS NULL AND expires_at > $2
		 RETURNING email`, d[:], at).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Email{}, app.ErrLoginLinkInvalid
	}
	if err != nil {
		return domain.Email{}, fmt.Errorf("use login request: %w", err)
	}
	to, err := domain.ParseEmail(raw)
	if err != nil {
		return domain.Email{}, fmt.Errorf("login request: stored address: %w", err)
	}
	return to, nil
}

// PurgeLoginRequests deletes the login requests whose link was used, or
// else expired, before cutoff and returns how many it deleted.
func (k *Keys) PurgeLoginRequests(ctx context.Context, cutoff time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	tag, err := k.pool.Exec(ctx, "DELETE FROM login_requests WHERE COALESCE(used_at, expires_at) < $1", cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete login requests: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ReplaceAccountKey revokes at r.CreatedAt every active key of the account
// r.Email.Account and stores r, in one transaction. A
// transaction lock on the account serialises two replacements of the same
// account, so that it keeps one active key. It implements
// app.AccountKeyIssuer.
func (k *Keys) ReplaceAccountKey(ctx context.Context, r domain.KeyRecord) (domain.KeyRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, k.pool, func(tx pgx.Tx) error {
		account := r.Email.Account()
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", account); err != nil {
			return fmt.Errorf("lock account: %w", err)
		}
		if _, err := tx.Exec(ctx,
			// GREATEST: the clocks of two instances may differ.
			"UPDATE api_keys SET revoked_at = GREATEST(created_at, $2) WHERE account = $1 AND revoked_at IS NULL",
			account, r.CreatedAt); err != nil {
			return fmt.Errorf("revoke account keys: %w", err)
		}
		return insertKey(ctx, tx, &r)
	})
	if err != nil {
		return domain.KeyRecord{}, fmt.Errorf("replace account key: %w", err)
	}
	return r, nil
}
