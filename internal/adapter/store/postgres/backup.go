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
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// A dump is a tar archive: manifest.json, then one <table>.csv per backed
// up table, in the order of backedUp. Each CSV file is the output of COPY
// with a header line, timestamps in ISO format and UTC (sessionSettings).
const (
	dumpFormat   = 1
	manifestName = "manifest.json"
)

// Bounds of a dump, far above the size of the key store, a few hundred
// bytes a key: about 100,000 keys a table. A backup holds the table, its
// archive and its sealed copy in memory, within the 256 MB of the backup
// container (deploy/tofu/backup.tf).
const (
	maxManifestBytes = 1 << 20
	maxTableBytes    = 32 << 20
)

// dumpTimeout bounds a whole dump.
const dumpTimeout = 2 * time.Minute

// sessionSettings make the CSV output of COPY the same whatever the
// configuration of the server, and readable by every server.
const sessionSettings = "SET LOCAL DateStyle = 'ISO, YMD'; SET LOCAL TimeZone = 'UTC'; SET LOCAL IntervalStyle = 'iso_8601'"

// table is a table of the backup, its columns in the order of its CSV file.
type table struct {
	name    string
	columns []string
	// identity is the identity column, whose sequence a restore moves past
	// the restored rows.
	identity string
}

// csvName is the name of the file of t in a dump.
func (t table) csvName() string { return t.name + ".csv" }

// backedUp are the tables a backup copies (#75). Their columns are part of
// the dump format: a migration that changes them needs a new dumpFormat, as
// HEADER MATCH refuses a dump whose columns differ.
var backedUp = []table{{
	name:     "api_keys",
	columns:  []string{"id", "digest", "environment", "label", "hint", "email", "account", "created_at", "revoked_at"},
	identity: "id",
}}

// leftOut are the tables a backup leaves out on purpose: the login
// requests live for minutes and a lost one is sent again (data
// minimisation); schema_migrations is rebuilt by the migrations.
var leftOut = []string{"login_requests", "schema_migrations"}

// ClassifiedTables returns every table a backup copies or leaves out on
// purpose. A table of the schema missing from it is a migration that did
// not say whether its rows are backed up.
func ClassifiedTables() []string {
	names := slices.Clone(leftOut)
	for _, t := range backedUp {
		names = append(names, t.name)
	}
	return names
}

// manifest describes a dump.
type manifest struct {
	Format        int             `json:"format"`
	SchemaVersion uint            `json:"schema_version"`
	Tables        []manifestTable `json:"tables"`
}

// manifestTable is a table of a dump and its number of rows.
type manifestTable struct {
	Name string `json:"name"`
	Rows int64  `json:"rows"`
}

// Dump writes a dump of the backed-up tables to w, read in one read-only
// snapshot of the database: the role of the backup may only read (#75).
// It implements app.BackupDumper.
func (k *Keys) Dump(ctx context.Context, w io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, dumpTimeout)
	defer cancel()
	tx, err := k.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("dump: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // Read only: nothing to keep.
	m, files, err := copyTables(ctx, tx)
	if err != nil {
		return err
	}
	return writeDump(w, m, files)
}

// copyTables returns the manifest and the CSV files of the backed-up
// tables, read through tx.
func copyTables(ctx context.Context, tx pgx.Tx) (manifest, [][]byte, error) {
	version, err := schemaVersion(ctx, tx)
	if err != nil {
		return manifest{}, nil, err
	}
	if _, err := tx.Exec(ctx, sessionSettings); err != nil {
		return manifest{}, nil, fmt.Errorf("dump: session settings: %w", err)
	}
	m := manifest{Format: dumpFormat, SchemaVersion: version}
	files := make([][]byte, 0, len(backedUp))
	for _, t := range backedUp {
		var b bytes.Buffer
		sql := fmt.Sprintf("COPY (SELECT %s FROM %s ORDER BY %s) TO STDOUT WITH (FORMAT csv, HEADER true)",
			columnList(t), pgx.Identifier{t.name}.Sanitize(), columnList(t))
		tag, err := tx.Conn().PgConn().CopyTo(ctx, &limitWriter{w: &b, left: maxTableBytes}, sql)
		if err != nil {
			return manifest{}, nil, fmt.Errorf("dump %s: %w", t.name, err)
		}
		m.Tables = append(m.Tables, manifestTable{Name: t.name, Rows: tag.RowsAffected()})
		files = append(files, b.Bytes())
	}
	return m, files, nil
}

// schemaVersion returns the migration version of the schema, read through
// q; a schema left dirty by a failed migration is refused.
func schemaVersion(ctx context.Context, q querier) (uint, error) {
	var (
		version int64
		dirty   bool
	)
	if err := q.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
		return 0, fmt.Errorf("schema version: %w", err)
	}
	if dirty || version < 0 {
		return 0, fmt.Errorf("schema version %d: dirty, a migration failed", version)
	}
	return uint(version), nil
}

// columnList returns the quoted columns of t, comma-separated.
func columnList(t table) string {
	quoted := make([]string, len(t.columns))
	for i, c := range t.columns {
		quoted[i] = pgx.Identifier{c}.Sanitize()
	}
	return strings.Join(quoted, ", ")
}

// writeDump writes the tar archive of m and files, in the order of
// backedUp, to w.
func writeDump(w io.Writer, m manifest, files [][]byte) error {
	mb, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("dump: manifest: %w", err)
	}
	tw := tar.NewWriter(w)
	if err := writeEntry(tw, manifestName, mb); err != nil {
		return err
	}
	for i, t := range backedUp {
		if err := writeEntry(tw, t.csvName(), files[i]); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	return nil
}

// writeEntry writes the file name holding b to tw.
func writeEntry(tw *tar.Writer, name string, b []byte) error {
	h := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(b)), Format: tar.FormatPAX}
	if err := tw.WriteHeader(h); err != nil {
		return fmt.Errorf("dump %s: %w", name, err)
	}
	if _, err := tw.Write(b); err != nil {
		return fmt.Errorf("dump %s: %w", name, err)
	}
	return nil
}

// errTooLarge means a table of a dump is over maxTableBytes.
var errTooLarge = errors.New("table over the size limit of a dump")

// limitWriter writes to w until left bytes are written, then fails.
type limitWriter struct {
	w    io.Writer
	left int64
}

func (l *limitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > l.left {
		return 0, errTooLarge
	}
	l.left -= int64(len(p))
	return l.w.Write(p)
}
