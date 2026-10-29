// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/text/encoding/charmap"

	"github.com/arnakoff/api/internal/platform/csvfile"
)

// ErrFormat means a file is not laid out as docs/sources.md records it.
var ErrFormat = errors.New("unexpected ARCEP file format")

// frenchDate is the JJ/MM/AAAA layout of the files' dates.
const frenchDate = "02/01/2006"

// newReader returns a reader of the Windows-1252 CSV r, decoded to UTF-8,
// after checking that its header is want.
func newReader(r io.Reader, want []string) (*csvfile.Reader, error) {
	return csvfile.Open(charmap.Windows1252.NewDecoder().Reader(r), want, ErrFormat)
}

// parseFrenchDate reads the JJ/MM/AAAA date of the named column.
func parseFrenchDate(column, s string) (time.Time, error) {
	t, err := time.Parse(frenchDate, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s not JJ/MM/AAAA", ErrFormat, column)
	}
	return t, nil
}

// errEmpty reports a required column left empty.
func errEmpty(column string) error {
	return fmt.Errorf("%w: %s empty", ErrFormat, column)
}
