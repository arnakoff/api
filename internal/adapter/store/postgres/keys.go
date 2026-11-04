// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// queryTimeout bounds every query, on top of the deadline of its context.
const queryTimeout = 3 * time.Second

// keyColumns are the columns scanned by scanKey, in its order.
const keyColumns = "id, digest, environment, label, hint, email, created_at, revoked_at"

// Keys keeps the records of API keys in the api_keys table, and the login
// requests in the login_requests table. It implements app.KeyFinder,
// app.KeyIssuer, app.KeyPurger, app.LoginRequestStore and
// app.AccountKeyIssuer.
type Keys struct {
	pool *pgxpool.Pool
}

// Open returns a Keys on the database of conn, which ParseConfig checks.
// It does not connect: the first query does, so the API starts and answers
// 503 while the database is unreachable.
func Open(ctx context.Context, conn string) (*Keys, error) {
	cfg, err := ParseConfig(conn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	return &Keys{pool: pool}, nil
}

// Close closes every connection, waiting for those in use.
func (k *Keys) Close() {
	k.pool.Close()
}

// FindKey returns the record of the key of digest d, or app.ErrKeyNotFound.
func (k *Keys) FindKey(ctx context.Context, d domain.KeyDigest) (domain.KeyRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	row := k.pool.QueryRow(ctx, "SELECT "+keyColumns+" FROM api_keys WHERE digest = $1", d[:])
	rec, err := scanKey(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.KeyRecord{}, app.ErrKeyNotFound
	}
	return rec, err
}

// CreateKey stores r and returns it with its identifier set.
func (k *Keys) CreateKey(ctx context.Context, r domain.KeyRecord) (domain.KeyRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	if err := insertKey(ctx, k.pool, &r); err != nil {
		return domain.KeyRecord{}, err
	}
	return r, nil
}

// querier runs a query returning a row: the pool or a transaction.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// insertKey stores r through q and sets its identifier.
func insertKey(ctx context.Context, q querier, r *domain.KeyRecord) error {
	err := q.QueryRow(ctx,
		`INSERT INTO api_keys (digest, environment, label, hint, email, account, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		r.Digest[:], r.Environment.String(), r.Label, r.Hint,
		optional(r.Email, domain.Email.String), optional(r.Email, domain.Email.Account), r.CreatedAt,
	).Scan(&r.ID)
	if err != nil {
		return fmt.Errorf("insert API key: %w", err)
	}
	return nil
}

// optional is the value of a column of e, of(e): NULL for no address.
func optional(e domain.Email, of func(domain.Email) string) any {
	if e.IsZero() {
		return nil
	}
	return of(e)
}

// ListKeys returns every record, oldest first.
func (k *Keys) ListKeys(ctx context.Context) ([]domain.KeyRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	rows, err := k.pool.Query(ctx, "SELECT "+keyColumns+" FROM api_keys ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list API keys: %w", err)
	}
	recs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.KeyRecord, error) {
		return scanKey(row)
	})
	if err != nil {
		return nil, fmt.Errorf("list API keys: %w", err)
	}
	return recs, nil
}

// RevokeKey marks the key id revoked at at, or returns app.ErrKeyNotFound.
// A key already revoked keeps its first revocation time.
func (k *Keys) RevokeKey(ctx context.Context, id int64, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	tag, err := k.pool.Exec(ctx,
		"UPDATE api_keys SET revoked_at = COALESCE(revoked_at, $2) WHERE id = $1", id, at)
	if err != nil {
		return fmt.Errorf("revoke API key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrKeyNotFound
	}
	return nil
}

// PurgeRevokedKeys deletes the keys revoked before cutoff and returns how
// many it deleted. Several instances may run it at once: each deletes what
// is left.
func (k *Keys) PurgeRevokedKeys(ctx context.Context, cutoff time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	tag, err := k.pool.Exec(ctx, "DELETE FROM api_keys WHERE revoked_at < $1", cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete revoked API keys: %w", err)
	}
	return tag.RowsAffected(), nil
}

// scanKey reads a row of keyColumns.
func scanKey(row pgx.Row) (domain.KeyRecord, error) {
	var (
		rec     domain.KeyRecord
		digest  []byte
		env     string
		email   *string
		revoked *time.Time
	)
	if err := row.Scan(&rec.ID, &digest, &env, &rec.Label, &rec.Hint, &email, &rec.CreatedAt, &revoked); err != nil {
		return domain.KeyRecord{}, err
	}
	if len(digest) != len(rec.Digest) {
		return domain.KeyRecord{}, fmt.Errorf("API key %d: digest of %d bytes", rec.ID, len(digest))
	}
	copy(rec.Digest[:], digest)
	e, err := domain.ParseKeyEnvironment(env)
	if err != nil {
		return domain.KeyRecord{}, fmt.Errorf("API key %d: %w", rec.ID, err)
	}
	rec.Environment = e
	if email != nil {
		if rec.Email, err = domain.ParseEmail(*email); err != nil {
			return domain.KeyRecord{}, fmt.Errorf("API key %d: %w", rec.ID, err)
		}
	}
	rec.CreatedAt = rec.CreatedAt.UTC()
	if revoked != nil {
		rec.RevokedAt = revoked.UTC()
	}
	return rec, nil
}
