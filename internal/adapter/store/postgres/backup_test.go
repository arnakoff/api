// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

var _ app.BackupDumper = (*postgres.Keys)(nil)

// seedKeys stores a test key, a revoked live key and an account key in
// keys, and a login request that a backup leaves out.
func seedKeys(t *testing.T, keys *postgres.Keys) []domain.KeyRecord {
	t.Helper()
	ctx := context.Background()
	revoked, err := keys.CreateKey(ctx, record(1, domain.KeyLive))
	if err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	if err := keys.RevokeKey(ctx, revoked.ID, created.Add(time.Hour)); err != nil {
		t.Fatalf("RevokeKey() error = %v", err)
	}
	if _, err := keys.CreateKey(ctx, record(2, domain.KeyTest)); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	jeanne := mustEmail(t, "jeanne.martin@example.org")
	if _, err := keys.ReplaceAccountKey(ctx, accountRecord(3, jeanne, created)); err != nil {
		t.Fatalf("ReplaceAccountKey() error = %v", err)
	}
	if err := keys.CreateLoginRequest(ctx, domain.NewLoginRequest(domain.TokenDigest{9}, jeanne, created)); err != nil {
		t.Fatalf("CreateLoginRequest() error = %v", err)
	}
	recs, err := keys.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	return recs
}

// dump returns the dump of keys.
func dump(t *testing.T, keys *postgres.Keys) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := keys.Dump(context.Background(), &b); err != nil {
		t.Fatalf("Dump() error = %v", err)
	}
	return b.Bytes()
}

func TestDumpRestoresIntoAnEmptyDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	keys := openKeys(t)
	want := seedKeys(t, keys)
	b := dump(t, keys)

	conn := testDatabase(t)
	if err := postgres.Restore(ctx, conn, bytes.NewReader(b)); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	restored, err := postgres.Open(ctx, conn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(restored.Close)
	// Restore applies the migrations after the dump's version.
	if err := postgres.Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate() after Restore error = %v", err)
	}
	got, err := restored.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("restored keys = %+v, want %+v", got, want)
	}
	// The identity sequence goes on after the restored rows.
	next, err := restored.CreateKey(ctx, record(4, domain.KeyTest))
	if err != nil {
		t.Fatalf("CreateKey() after Restore error = %v", err)
	}
	if last := want[len(want)-1].ID; next.ID <= last {
		t.Errorf("new key id = %d, want above the restored %d", next.ID, last)
	}
}

func TestDumpLeavesOutLoginRequests(t *testing.T) {
	t.Parallel()
	keys := openKeys(t)
	seedKeys(t, keys)
	names := entries(t, dump(t, keys))
	if want := []string{"manifest.json", "api_keys.csv"}; !slices.Equal(names, want) {
		t.Errorf("dump entries = %v, want %v", names, want)
	}
}

func TestRestoreRefusesAStoreWithKeys(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	keys := openKeys(t)
	seedKeys(t, keys)
	b := dump(t, keys)
	conn := testDatabase(t)
	if err := postgres.Restore(ctx, conn, bytes.NewReader(b)); err != nil {
		t.Fatalf("first Restore() error = %v", err)
	}
	if err := postgres.Restore(ctx, conn, bytes.NewReader(b)); !errors.Is(err, postgres.ErrRestoreNotEmpty) {
		t.Errorf("second Restore() error = %v, want ErrRestoreNotEmpty", err)
	}
}

func TestRestoreRefusesABrokenDump(t *testing.T) {
	t.Parallel()
	tests := map[string][]byte{
		"not a tar":      []byte("not a tar archive"),
		"no manifest":    tarOf(t, map[string]string{"api_keys.csv": "id\n"}),
		"other format":   tarOf(t, map[string]string{"manifest.json": `{"format":2,"schema_version":3,"tables":[]}`}),
		"missing table":  tarOf(t, map[string]string{"manifest.json": `{"format":1,"schema_version":3,"tables":[{"name":"api_keys","rows":0}]}`}),
		"unknown table":  tarOf(t, map[string]string{"manifest.json": `{"format":1,"schema_version":3,"tables":[{"name":"login_requests","rows":0}]}`, "login_requests.csv": ""}),
		"newer schema":   tarOf(t, map[string]string{"manifest.json": `{"format":1,"schema_version":999,"tables":[{"name":"api_keys","rows":0}]}`, "api_keys.csv": ""}),
		"row count lies": tarOf(t, map[string]string{"manifest.json": `{"format":1,"schema_version":3,"tables":[{"name":"api_keys","rows":5}]}`, "api_keys.csv": "id,digest,environment,label,hint,email,account,created_at,revoked_at\n"}),
	}
	for name, b := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			conn := testDatabase(t)
			if err := postgres.Restore(t.Context(), conn, bytes.NewReader(b)); !errors.Is(err, postgres.ErrRestoreInvalid) {
				t.Errorf("Restore() error = %v, want ErrRestoreInvalid", err)
			}
		})
	}
}

// Every table of the schema is either in the backup or left out on
// purpose: a migration that adds a table must say which.
func TestEveryTableIsClassifiedForTheBackup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	conn := testDatabase(t)
	if err := postgres.Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	c, err := pgx.Connect(ctx, conn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = c.Close(ctx) }()
	rows, err := c.Query(ctx, `SELECT table_name FROM information_schema.tables
		WHERE table_schema = current_schema() ORDER BY table_name`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	want := postgres.ClassifiedTables()
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("tables = %v, classified for the backup = %v", got, want)
	}
}

// entries returns the names in the tar archive b, in order.
func entries(t *testing.T, b []byte) []string {
	t.Helper()
	var names []string
	tr := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return names
		}
		if err != nil {
			t.Fatalf("read dump: %v", err)
		}
		names = append(names, h.Name)
	}
}

// tarOf returns a tar archive of files, manifest.json first.
func tarOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	slices.SortFunc(names, func(a, b string) int {
		switch {
		case a == "manifest.json":
			return -1
		case b == "manifest.json":
			return 1
		}
		return 0
	})
	for _, n := range names {
		body := files[n]
		if err := tw.WriteHeader(&tar.Header{Name: n, Mode: 0o600, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// The backup role of production may only read the tables (readonly).
func TestDumpNeedsOnlyToRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	conn := testDatabase(t)
	keys := openKeysAt(t, conn)
	seedKeys(t, keys)
	want := dump(t, keys)
	reader, err := postgres.Open(ctx, readOnlyRole(t, conn))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(reader.Close)
	if got := dump(t, reader); !bytes.Equal(got, want) {
		t.Errorf("dump by a read-only role differs:\n%s\nwant:\n%s", got, want)
	}
}

// readOnlyRole creates a login role that may only read the tables of the
// schema of conn, dropped at the end of t, and returns its connection URL.
func readOnlyRole(t *testing.T, conn string) string {
	t.Helper()
	ctx := context.Background()
	c, err := pgx.Connect(ctx, conn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = c.Close(ctx) }()
	var schema string
	if err := c.QueryRow(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
		t.Fatalf("current schema: %v", err)
	}
	role := schema + "_reader"
	ident, sch := pgx.Identifier{role}.Sanitize(), pgx.Identifier{schema}.Sanitize()
	for _, sql := range []string{
		"CREATE ROLE " + ident + " LOGIN PASSWORD 'reader'",
		"GRANT USAGE ON SCHEMA " + sch + " TO " + ident,
		"GRANT SELECT ON ALL TABLES IN SCHEMA " + sch + " TO " + ident,
	} {
		if _, err := c.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	t.Cleanup(func() {
		c, err := pgx.Connect(context.Background(), conn)
		if err != nil {
			t.Errorf("connect to drop role: %v", err)
			return
		}
		defer func() { _ = c.Close(context.Background()) }()
		if _, err := c.Exec(context.Background(), "DROP OWNED BY "+ident+"; DROP ROLE "+ident); err != nil {
			t.Errorf("drop role: %v", err)
		}
	})
	u, err := url.Parse(conn)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword(role, "reader")
	return u.String()
}
