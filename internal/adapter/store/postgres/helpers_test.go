// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
)

// envTestDatabase names the database the tests of this package run
// against, such as postgres://postgres:test@127.0.0.1:5432/arnakoff_test.
// Unset, they are skipped: go test never reaches a database unasked.
const envTestDatabase = "ARNAKOFF_TEST_DATABASE_URL"

// testDatabase returns the connection URL of a schema of its own for t,
// dropped at the end of the test, or skips t when envTestDatabase is unset.
func testDatabase(t *testing.T) string {
	t.Helper()
	base := os.Getenv(envTestDatabase)
	if base == "" {
		t.Skipf("%s is not set", envTestDatabase)
	}
	schema := "test_" + strings.ToLower(rand.Text())
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect to the test database: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	ident := pgx.Identifier{schema}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE SCHEMA "+ident); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		c, err := pgx.Connect(context.Background(), base)
		if err != nil {
			t.Errorf("connect to drop schema: %v", err)
			return
		}
		defer func() { _ = c.Close(context.Background()) }()
		if _, err := c.Exec(context.Background(), "DROP SCHEMA "+ident+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})
	return withSearchPath(t, base, schema)
}

// withSearchPath returns base with schema as its search_path, which pgx
// sends as a run-time parameter.
func withSearchPath(t *testing.T, base, schema string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("%s is not a URL", envTestDatabase)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

// terminateListeners ends, from another connection, every watch of
// revocations connected to the test database.
func terminateListeners(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv(envTestDatabase))
	if err != nil {
		t.Fatalf("connect to the test database: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity
		 WHERE datname = current_database() AND application_name = $1`, postgres.WatchApplicationName); err != nil {
		t.Fatalf("terminate listeners: %v", err)
	}
}

// notify sends payload on the channel of the revocations of the test
// database.
func notify(t *testing.T, payload string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv(envTestDatabase))
	if err != nil {
		t.Fatalf("connect to the test database: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, "SELECT pg_notify('api_key_revoked', $1)", payload); err != nil {
		t.Fatalf("notify: %v", err)
	}
}
