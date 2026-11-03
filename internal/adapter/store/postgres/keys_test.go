// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// The store implements the ports of the use cases.
var (
	_ app.KeyFinder = (*postgres.Keys)(nil)
	_ app.KeyIssuer = (*postgres.Keys)(nil)
	_ app.KeyPurger = (*postgres.Keys)(nil)
)

// created is the issue time of the records of these tests. PostgreSQL
// keeps microseconds.
var created = time.Date(2026, 11, 2, 10, 30, 0, 123456000, time.UTC)

// openKeys migrates a schema of its own for t and opens the store on it.
func openKeys(t *testing.T) *postgres.Keys {
	t.Helper()
	conn := testDatabase(t)
	if err := postgres.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	keys, err := postgres.Open(context.Background(), conn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(keys.Close)
	return keys
}

// record returns a record of digest {b} issued at created.
func record(b byte, env domain.KeyEnvironment) domain.KeyRecord {
	return domain.KeyRecord{
		Digest: domain.KeyDigest{b}, Environment: env, Label: "Bot Discord", Hint: "a1B2", CreatedAt: created,
	}
}

func TestKeysCreateAndFind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	rec, err := keys.CreateKey(ctx, record(1, domain.KeyTest))
	if err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	if rec.ID == 0 {
		t.Error("CreateKey() left the identifier unset")
	}
	got, err := keys.FindKey(ctx, domain.KeyDigest{1})
	if err != nil {
		t.Fatalf("FindKey() error = %v", err)
	}
	if !got.CreatedAt.Equal(created) || got.CreatedAt.Location() != time.UTC {
		t.Errorf("FindKey() CreatedAt = %v, want %v in UTC", got.CreatedAt, created)
	}
	got.CreatedAt = rec.CreatedAt
	if got != rec {
		t.Errorf("FindKey() = %+v, want %+v", got, rec)
	}
	if _, err := keys.FindKey(ctx, domain.KeyDigest{2}); !errors.Is(err, app.ErrKeyNotFound) {
		t.Errorf("FindKey(unknown) error = %v, want ErrKeyNotFound", err)
	}
}

func TestKeysCreateRejectsDuplicateDigest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	if _, err := keys.CreateKey(ctx, record(1, domain.KeyLive)); err != nil {
		t.Fatal(err)
	}
	if _, err := keys.CreateKey(ctx, record(1, domain.KeyLive)); err == nil {
		t.Error("CreateKey() of a digest already stored: no error")
	}
}

func TestKeysListAndRevoke(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	first, err := keys.CreateKey(ctx, record(1, domain.KeyLive))
	if err != nil {
		t.Fatal(err)
	}
	second, err := keys.CreateKey(ctx, record(2, domain.KeyTest))
	if err != nil {
		t.Fatal(err)
	}
	revoked := created.Add(time.Hour)
	if err := keys.RevokeKey(ctx, second.ID, revoked); err != nil {
		t.Fatalf("RevokeKey() error = %v", err)
	}
	if err := keys.RevokeKey(ctx, second.ID, revoked.Add(time.Hour)); err != nil {
		t.Fatalf("RevokeKey() again error = %v", err)
	}
	if err := keys.RevokeKey(ctx, second.ID+100, revoked); !errors.Is(err, app.ErrKeyNotFound) {
		t.Errorf("RevokeKey(unknown) error = %v, want ErrKeyNotFound", err)
	}
	recs, err := keys.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if len(recs) != 2 || recs[0].ID != first.ID || !recs[0].Active() || !recs[1].RevokedAt.Equal(revoked) {
		t.Errorf("ListKeys() = %+v, want first active, then second revoked at %v once", recs, revoked)
	}
}

func TestKeysUnreachable(t *testing.T) {
	t.Parallel()

	// Nothing listens on port 1 of the loopback address.
	keys, err := postgres.Open(context.Background(), "postgres://api@127.0.0.1:1/arnakoff?connect_timeout=1")
	if err != nil {
		t.Fatalf("Open() error = %v, want a lazy pool", err)
	}
	defer keys.Close()
	_, err = keys.FindKey(context.Background(), domain.KeyDigest{1})
	if err == nil || errors.Is(err, app.ErrKeyNotFound) {
		t.Errorf("FindKey() on an unreachable database error = %v, want a connection error", err)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	t.Parallel()

	conn := testDatabase(t)
	for range 2 {
		if err := postgres.Migrate(context.Background(), conn); err != nil {
			t.Fatalf("Migrate() error = %v", err)
		}
	}
}

func TestOpenAndMigrateRejectInsecure(t *testing.T) {
	t.Parallel()

	conn := "postgres://api:pw@db.example.eu/arnakoff?sslmode=require" //nolint:gosec // G101: a fake password.
	if _, err := postgres.Open(context.Background(), conn); !errors.Is(err, postgres.ErrInsecure) {
		t.Errorf("Open() error = %v, want ErrInsecure", err)
	}
	if err := postgres.Migrate(context.Background(), conn); !errors.Is(err, postgres.ErrInsecure) {
		t.Errorf("Migrate() error = %v, want ErrInsecure", err)
	}
}

func TestKeysPurgeRevokedKeysDeletesOnlyKeysRevokedBeforeTheCutoff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	cutoff := created.Add(48 * time.Hour)
	revocations := map[byte]time.Time{
		2: cutoff.Add(-time.Microsecond), // before: deleted
		3: cutoff,                        // at: kept
		4: cutoff.Add(time.Hour),         // after: kept
	}
	for b := byte(1); b <= 4; b++ {
		rec, err := keys.CreateKey(ctx, record(b, domain.KeyLive))
		if err != nil {
			t.Fatalf("CreateKey() error = %v", err)
		}
		if at, ok := revocations[b]; ok {
			if err := keys.RevokeKey(ctx, rec.ID, at); err != nil {
				t.Fatalf("RevokeKey() error = %v", err)
			}
		}
	}

	n, err := keys.PurgeRevokedKeys(ctx, cutoff)
	if err != nil || n != 1 {
		t.Fatalf("PurgeRevokedKeys() = %d, %v; want 1 key deleted", n, err)
	}
	if _, err := keys.FindKey(ctx, domain.KeyDigest{2}); !errors.Is(err, app.ErrKeyNotFound) {
		t.Errorf("FindKey(purged) error = %v, want ErrKeyNotFound", err)
	}
	for _, b := range []byte{1, 3, 4} {
		if _, err := keys.FindKey(ctx, domain.KeyDigest{b}); err != nil {
			t.Errorf("FindKey(%d) error = %v, want the key kept", b, err)
		}
	}
}
