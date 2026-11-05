// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"io"
)

// Restorer loads dumps into the database of its connection string with
// Restore. It implements app.KeyDumpLoader.
type Restorer struct {
	conn string
	opts []Option
}

// NewRestorer returns a Restorer into the database of conn, checked with
// opts. Nothing connects until a dump is loaded.
func NewRestorer(conn string, opts ...Option) *Restorer {
	return &Restorer{conn: conn, opts: opts}
}

// LoadDump restores the dump read from r, as Restore does.
func (r *Restorer) LoadDump(ctx context.Context, dump io.Reader) error {
	return Restore(ctx, r.conn, dump, r.opts...)
}
