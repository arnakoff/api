// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/arnakoff/api/internal/platform/csvfile"
)

// ErrFormat means a file is not laid out as docs/sources.md records it.
// Errors never quote a value of the file: it may hold an address.
var ErrFormat = errors.New("unexpected AMF file format")

// bom is the UTF-8 byte order mark the AMF files start with.
var bom = []byte("\xef\xbb\xbf")

// newReader returns a reader of the AMF CSV r, past its byte order mark if
// any, after checking that its header is want.
func newReader(r io.Reader, want []string) (*csvfile.Reader, error) {
	br := bufio.NewReader(r)
	if b, err := br.Peek(len(bom)); err == nil && bytes.Equal(b, bom) {
		_, _ = br.Discard(len(bom))
	}
	return csvfile.Open(br, want, ErrFormat)
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
