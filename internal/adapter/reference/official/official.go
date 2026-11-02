// SPDX-License-Identifier: AGPL-3.0-only

// Package official reads the reference list of official domains often
// imitated, shipped with the binary (domains.csv, documented in
// docs/sources.md).
package official

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/csvfile"
)

// Source names the reference list among the sources.
const Source = signal.LookalikeSource

// ErrFormat means a reference list departs from its format. The error
// locates the line.
var ErrFormat = errors.New("invalid official domains file")

// header is the first line of a reference list.
var header = []string{"domain", "alternates", "names", "of", "proof"}

//go:embed domains.csv
var shipped []byte

// Load reads the reference list shipped with the binary.
func Load() (signal.Officials, error) {
	return Parse(bytes.NewReader(shipped))
}

// Parse reads a reference list: a CSV file with header, one official site
// per row, alternates and names separated by "|". A malformed file is
// rejected with ErrFormat, a malformed row with signal.ErrInvalidOfficial.
func Parse(r io.Reader) (signal.Officials, error) {
	cr, err := csvfile.OpenSeparated(r, ',', header, ErrFormat)
	if err != nil {
		return signal.Officials{}, err
	}
	var rows []signal.Official
	err = cr.Each(func(rec []string) error {
		if len(rec) != len(header) {
			return fmt.Errorf("%w: %d fields, want %d", ErrFormat, len(rec), len(header))
		}
		rows = append(rows, signal.Official{
			Domain:     rec[0],
			Alternates: list(rec[1]),
			Names:      list(rec[2]),
			Of:         rec[3],
			Proof:      rec[4],
		})
		return nil
	})
	if err != nil {
		return signal.Officials{}, err
	}
	return signal.NewOfficials(rows)
}

// list splits a "|"-separated field; an empty field is an empty list.
func list(field string) []string {
	if field == "" {
		return nil
	}
	return strings.Split(field, "|")
}
