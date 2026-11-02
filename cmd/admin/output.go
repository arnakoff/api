// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// writeIssued writes a new key to w: the only time it is ever shown.
func writeIssued(w io.Writer, key domain.APIKey, rec domain.KeyRecord) error {
	_, err := fmt.Fprintf(w, "key %d issued; it is shown once, store it now:\n%s\n", rec.ID, key)
	return err
}

// writeKeys writes recs to w as a table, one key a line.
func writeKeys(w io.Writer, recs []domain.KeyRecord) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ID\tENV\tHINT\tCREATED\tREVOKED\tLABEL")
	for _, r := range recs {
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.Environment, r.Hint, stamp(r.CreatedAt), stamp(r.RevokedAt), r.Label)
	}
	return tw.Flush()
}

// stamp writes t in RFC 3339, or "-" when it is zero.
func stamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}
