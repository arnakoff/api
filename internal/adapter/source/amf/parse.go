// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Source is the name of the AMF blacklist, as shown to users.
const Source domain.Source = "AMF"

// Columns of a snapshot row, in order.
const (
	// ColumnKind is "domain" or "email".
	ColumnKind = "kind"
	// ColumnKey is the normalised host or address the entry is looked up by.
	ColumnKey = "key"
	// ColumnPath is the path and query of a site entry that targets one
	// page, "" when it targets the whole site.
	ColumnPath = "path"
	// ColumnEntry is the entry as the AMF lists it.
	ColumnEntry = "entry"
	// ColumnCategories are the AMF categories, separated by "|".
	ColumnCategories = "categories"
	// ColumnListedOn is the listing date, AAAA-MM-JJ.
	ColumnListedOn = "listed_on"
)

var columns = []string{ColumnKind, ColumnKey, ColumnPath, ColumnEntry, ColumnCategories, ColumnListedOn}

// upstreamHeader is the header of the published file (docs/sources.md).
var upstreamHeader = []string{"nom", "categorie", "date_inscription"}

// dateLayout is the layout of date_inscription.
const dateLayout = time.DateOnly

// ErrFormat means the file is not laid out as docs/sources.md records it.
var ErrFormat = errors.New("unexpected AMF file format")

// bom is the UTF-8 byte order mark the file starts with.
var bom = []byte("\xef\xbb\xbf")

// Parse reads the published CSV file into a snapshot, one row per site or
// address. Free-text names are left out. A file that departs from the
// recorded format is rejected with ErrFormat rather than half read.
func Parse(r io.Reader) (domain.Snapshot, error) {
	cr := newReader(r)
	head, err := cr.Read()
	if err != nil || !slices.Equal(head, upstreamHeader) {
		// Values are left out of errors: they may hold an address.
		return domain.Snapshot{}, fmt.Errorf("%w: header of %d fields", ErrFormat, len(head))
	}
	snap := domain.Snapshot{Source: Source, Header: columns}
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return snap, nil
		}
		if err != nil {
			return domain.Snapshot{}, fmt.Errorf("%w: %w", ErrFormat, err)
		}
		if err := addRow(&snap, rec); err != nil {
			line, _ := cr.FieldPos(0)
			return domain.Snapshot{}, fmt.Errorf("line %d: %w", line, err)
		}
	}
}

// newReader returns a CSV reader of r past its byte order mark, if any.
func newReader(r io.Reader) *csv.Reader {
	br := bufio.NewReader(r)
	if b, err := br.Peek(len(bom)); err == nil && bytes.Equal(b, bom) {
		_, _ = br.Discard(len(bom))
	}
	cr := csv.NewReader(br)
	cr.Comma = ';'
	cr.ReuseRecord = true
	return cr
}

// addRow appends the entry rec to snap, unless it is a free-text name, and
// moves snap.Newest to its date.
func addRow(snap *domain.Snapshot, rec []string) error {
	nom, categories, listed := rec[0], rec[1], rec[2]
	if categories == "" {
		return fmt.Errorf("%w: no category", ErrFormat)
	}
	date, err := time.Parse(dateLayout, listed)
	if err != nil {
		return fmt.Errorf("%w: date_inscription not AAAA-MM-JJ", ErrFormat)
	}
	kind, key, path, ok := classify(nom)
	if !ok {
		return nil
	}
	snap.Rows = append(snap.Rows, []string{kind, key, path, nom, categories, listed})
	if date.After(snap.Newest) {
		snap.Newest = date
	}
	return nil
}
