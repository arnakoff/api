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
)

// ErrFormat means a file is not laid out as docs/sources.md records it.
// Errors never quote a value of the file: it may hold an address.
var ErrFormat = errors.New("unexpected AMF file format")

// bom is the UTF-8 byte order mark the AMF files start with.
var bom = []byte("\xef\xbb\xbf")

// newReader returns a reader of the AMF CSV r, past its byte order mark if
// any, after checking that its header is want.
func newReader(r io.Reader, want []string) (*csv.Reader, error) {
	br := bufio.NewReader(r)
	if b, err := br.Peek(len(bom)); err == nil && bytes.Equal(b, bom) {
		_, _ = br.Discard(len(bom))
	}
	cr := csv.NewReader(br)
	cr.Comma = ';'
	cr.ReuseRecord = true
	head, err := cr.Read()
	if err != nil || !slices.Equal(head, want) {
		return nil, fmt.Errorf("%w: header of %d fields", ErrFormat, len(head))
	}
	return cr, nil
}

// eachRecord calls add with every record after the header, and stops at the
// first error, located by line.
func eachRecord(cr *csv.Reader, add func(rec []string) error) error {
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFormat, err)
		}
		if err := add(rec); err != nil {
			line, _ := cr.FieldPos(0)
			return fmt.Errorf("line %d: %w", line, err)
		}
	}
}

// parseDate reads the AAAA-MM-JJ date of the named column.
func parseDate(column, s string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s not AAAA-MM-JJ", ErrFormat, column)
	}
	return t, nil
}

// errEmpty reports a required column left empty.
func errEmpty(column string) error {
	return fmt.Errorf("%w: %s empty", ErrFormat, column)
}

// errUnknown reports a value a column is not known to take.
func errUnknown(column string) error {
	return fmt.Errorf("%w: %s takes an unknown value", ErrFormat, column)
}
