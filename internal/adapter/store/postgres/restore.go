// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
)

// ErrRestoreInvalid means a dump cannot be restored: not a dump of this
// format, incomplete, or of a schema newer than the migrations of this
// binary.
var ErrRestoreInvalid = errors.New("invalid dump")

// ErrRestoreNotEmpty means the database to restore into already holds keys,
// or a schema newer than the dump's: a restore never merges nor overwrites.
var ErrRestoreNotEmpty = errors.New("restore target is not an empty key store")

// restoreTimeout bounds the loading of a dump, once the schema is migrated.
const restoreTimeout = 5 * time.Minute

// dump is a dump read back: its manifest and the CSV file of each table,
// in the order of backedUp.
type dump struct {
	manifest manifest
	files    [][]byte
}

// Restore loads the dump read from r into the database of conn, checked
// with opts: it migrates the schema up to the dump's version, loads the
// rows in one transaction, then applies the remaining migrations. Run it
// with the role that applies the migrations. The tables of the dump must be
// empty.
func Restore(ctx context.Context, conn string, r io.Reader, opts ...Option) error {
	d, err := readDump(r)
	if err != nil {
		return err
	}
	cfg, err := ParseConfig(conn, opts...)
	if err != nil {
		return err
	}
	m, err := openMigrate(ctx, cfg.ConnConfig)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if err := migrateTo(m, d.manifest.SchemaVersion); err != nil {
		return err
	}
	if err := loadDump(ctx, cfg.ConnConfig, d); err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("restore: migrate up: %w", err)
	}
	return nil
}

// migrateTo migrates m up to version, the version of a dump. A schema
// already past it holds a newer store: ErrRestoreNotEmpty.
func migrateTo(m *migrate.Migrate, version uint) error {
	current, dirty, err := m.Version()
	switch {
	case errors.Is(err, migrate.ErrNilVersion):
	case err != nil:
		return fmt.Errorf("restore: schema version: %w", err)
	case dirty:
		return fmt.Errorf("restore: schema version %d is dirty, a migration failed", current)
	case current > version:
		return fmt.Errorf("%w: schema version %d is newer than the dump's %d", ErrRestoreNotEmpty, current, version)
	case current == version:
		return nil
	}
	if err := m.Migrate(version); err != nil {
		return fmt.Errorf("restore: migrate to %d: %w", version, err)
	}
	return nil
}

// loadDump loads d into the database of cc in one transaction.
func loadDump(ctx context.Context, cc *pgx.ConnConfig, d dump) error {
	ctx, cancel := context.WithTimeout(ctx, restoreTimeout)
	defer cancel()
	c, err := pgx.ConnectConfig(ctx, cc)
	if err != nil {
		return fmt.Errorf("restore: connect: %w", err)
	}
	defer func() { _ = c.Close(context.WithoutCancel(ctx)) }()
	return pgx.BeginFunc(ctx, c, func(tx pgx.Tx) error { return load(ctx, tx, d) })
}

// load copies the tables of d into empty tables through tx, then moves
// their identity sequences past the restored rows.
func load(ctx context.Context, tx pgx.Tx, d dump) error {
	if _, err := tx.Exec(ctx, sessionSettings); err != nil {
		return fmt.Errorf("restore: session settings: %w", err)
	}
	for i, t := range backedUp {
		if err := loadTable(ctx, tx, t, d.files[i], d.manifest.Tables[i].Rows); err != nil {
			return err
		}
	}
	return nil
}

// loadTable copies the CSV file b of t, which holds rows rows, into t,
// which must be empty.
func loadTable(ctx context.Context, tx pgx.Tx, t table, b []byte, rows int64) error {
	name := pgx.Identifier{t.name}.Sanitize()
	var used bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT FROM "+name+")").Scan(&used); err != nil {
		return fmt.Errorf("restore %s: %w", t.name, err)
	}
	if used {
		return fmt.Errorf("%w: %s has rows", ErrRestoreNotEmpty, t.name)
	}
	// HEADER MATCH refuses a file whose columns are not those of t, in its
	// order.
	sql := fmt.Sprintf("COPY %s (%s) FROM STDIN WITH (FORMAT csv, HEADER MATCH)", name, columnList(t))
	tag, err := tx.Conn().PgConn().CopyFrom(ctx, bytes.NewReader(b), sql)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrRestoreInvalid, t.name, err)
	}
	if tag.RowsAffected() != rows {
		return fmt.Errorf("%w: %s: %d rows, the manifest says %d", ErrRestoreInvalid, t.name, tag.RowsAffected(), rows)
	}
	if t.identity == "" {
		return nil
	}
	col := pgx.Identifier{t.identity}.Sanitize()
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		"SELECT setval(pg_get_serial_sequence($1, $2), COALESCE(MAX(%s), 1), MAX(%s) IS NOT NULL) FROM %s", col, col, name),
		t.name, t.identity); err != nil {
		return fmt.Errorf("restore %s: identity: %w", t.name, err)
	}
	return nil
}

// readDump reads and checks a dump: its manifest, then the CSV file of
// each backed-up table in order, and nothing else.
func readDump(r io.Reader) (dump, error) {
	tr := tar.NewReader(r)
	m, err := readManifest(tr)
	if err != nil {
		return dump{}, err
	}
	d := dump{manifest: m, files: make([][]byte, len(backedUp))}
	for i, t := range backedUp {
		if d.files[i], err = readNamed(tr, t.csvName(), maxTableBytes); err != nil {
			return dump{}, err
		}
	}
	if _, err := tr.Next(); !errors.Is(err, io.EOF) {
		return dump{}, fmt.Errorf("%w: entries after the last table", ErrRestoreInvalid)
	}
	return d, nil
}

// readManifest reads the manifest, the first entry of tr, and checks it.
func readManifest(tr *tar.Reader) (manifest, error) {
	b, err := readNamed(tr, manifestName, maxManifestBytes)
	if err != nil {
		return manifest{}, err
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return manifest{}, fmt.Errorf("%w: manifest: %w", ErrRestoreInvalid, err)
	}
	return m, checkManifest(m)
}

// checkManifest checks that m is of this format, lists the backed-up tables
// in their order and names a schema version this binary can migrate to.
func checkManifest(m manifest) error {
	if m.Format != dumpFormat {
		return fmt.Errorf("%w: format %d, want %d", ErrRestoreInvalid, m.Format, dumpFormat)
	}
	if len(m.Tables) != len(backedUp) {
		return fmt.Errorf("%w: %d tables, want %d", ErrRestoreInvalid, len(m.Tables), len(backedUp))
	}
	for i, t := range backedUp {
		if m.Tables[i].Name != t.name {
			return fmt.Errorf("%w: table %d is %q, want %s", ErrRestoreInvalid, i, m.Tables[i].Name, t.name)
		}
	}
	latest, err := latestVersion()
	if err != nil {
		return err
	}
	if m.SchemaVersion == 0 || m.SchemaVersion > latest {
		return fmt.Errorf("%w: schema version %d, this binary migrates up to %d", ErrRestoreInvalid, m.SchemaVersion, latest)
	}
	return nil
}

// readNamed reads the next entry of tr, which must be the regular file
// name, of at most limit bytes.
func readNamed(tr *tar.Reader, name string, limit int64) ([]byte, error) {
	h, err := tr.Next()
	if err != nil {
		return nil, fmt.Errorf("%w: want %s: %w", ErrRestoreInvalid, name, err)
	}
	if h.Name != name || h.Typeflag != tar.TypeReg || h.Size < 0 || h.Size > limit {
		return nil, fmt.Errorf("%w: entry %q, want the regular file %s of at most %d bytes", ErrRestoreInvalid, h.Name, name, limit)
	}
	b, err := io.ReadAll(io.LimitReader(tr, h.Size))
	if err != nil {
		return nil, fmt.Errorf("%w: entry %q: %w", ErrRestoreInvalid, name, err)
	}
	return b, nil
}
