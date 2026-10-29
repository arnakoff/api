// SPDX-License-Identifier: AGPL-3.0-only

package csvfile

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
)

// Reader reads the records of a file after its header.
type Reader struct {
	cr        *csv.Reader
	errFormat error
}

// Open checks that the semicolon-separated r starts with the header want
// and returns a reader of the records after it. A departure from the
// format is reported with errFormat, the source's own error.
func Open(r io.Reader, want []string, errFormat error) (*Reader, error) {
	cr := csv.NewReader(r)
	cr.Comma = ';'
	cr.ReuseRecord = true
	head, err := cr.Read()
	if err != nil || !slices.Equal(head, want) {
		return nil, fmt.Errorf("%w: header of %d fields", errFormat, len(head))
	}
	return &Reader{cr: cr, errFormat: errFormat}, nil
}

// Each calls add with every record, which add must not keep: the next one
// reuses it. It stops at the first error, located by line.
func (r *Reader) Each(add func(rec []string) error) error {
	for {
		rec, err := r.cr.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%w: %w", r.errFormat, err)
		}
		if err := add(rec); err != nil {
			line, _ := r.cr.FieldPos(0)
			return fmt.Errorf("line %d: %w", line, err)
		}
	}
}
