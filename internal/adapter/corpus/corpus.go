// SPDX-License-Identifier: AGPL-3.0-only

// Package corpus reads the labelled evaluation corpus of testdata/corpus:
// scam.csv, inputs an authority blacklisted, and legit.csv, inputs of
// verified legitimate entities.
//
// Both files are UTF-8, comma-separated, with the header
// input,type,label,added_at,note: the input as a user would paste it, its
// kind as domain.Kind writes it ("domain", "email"…), its label ("scam" or
// "legit", the one its file holds), the ISO date it was blacklisted or
// verified, and a free note citing where it comes from.
package corpus

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/csvfile"
)

// ErrFormat means a corpus file departs from its format. The error locates
// the line, never quotes a value.
var ErrFormat = errors.New("invalid corpus file")

// columns is the header of a corpus file.
var columns = []string{"input", "type", "label", "added_at", "note"}

// Files of a corpus directory and the label each holds, in reading order.
var files = []struct {
	name  string
	label app.Label
}{
	{"scam.csv", app.LabelScam},
	{"legit.csv", app.LabelLegit},
}

// Load reads scam.csv then legit.csv from dir. A case labelled otherwise
// than its file is rejected with ErrFormat.
func Load(dir string) ([]app.Case, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open corpus: %w", err)
	}
	defer func() { _ = root.Close() }() // Read only: a close error changes nothing.
	var all []app.Case
	for _, f := range files {
		cases, err := loadFile(root, f.name)
		if err != nil {
			return nil, err
		}
		for _, c := range cases {
			if c.Label != f.label {
				return nil, fmt.Errorf("%s: %w: a %s case", f.name, ErrFormat, c.Label)
			}
		}
		all = append(all, cases...)
	}
	return all, nil
}

// loadFile reads the corpus file name of root.
func loadFile(root *os.Root, name string) ([]app.Case, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open corpus file: %w", err)
	}
	defer func() { _ = f.Close() }() // Read only: a close error changes nothing.
	cases, err := Read(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return cases, nil
}

// Read reads the cases of one corpus file.
func Read(r io.Reader) ([]app.Case, error) {
	cr, err := csvfile.OpenSeparated(r, ',', columns, ErrFormat)
	if err != nil {
		return nil, err
	}
	var cases []app.Case
	err = cr.Each(func(rec []string) error {
		c, err := parseCase(rec)
		if err != nil {
			return err
		}
		cases = append(cases, c)
		return nil
	})
	return cases, err
}

// parseCase parses one record of a corpus file.
func parseCase(rec []string) (app.Case, error) {
	if len(rec) != len(columns) {
		return app.Case{}, fmt.Errorf("%w: %d fields, want %d", ErrFormat, len(rec), len(columns))
	}
	kind, err := domain.ParseKind(rec[1])
	if err != nil {
		return app.Case{}, fmt.Errorf("%w: %w", ErrFormat, err)
	}
	// The parse error is left out: it may quote the input.
	in, err := domain.ParseInput(rec[0], kind)
	if err != nil {
		return app.Case{}, fmt.Errorf("%w: input is not a valid %s", ErrFormat, kind)
	}
	label := app.Label(rec[2])
	if label != app.LabelScam && label != app.LabelLegit {
		return app.Case{}, fmt.Errorf("%w: unknown label", ErrFormat)
	}
	if _, err := time.Parse(time.DateOnly, rec[3]); err != nil {
		return app.Case{}, fmt.Errorf("%w: added_at is not an ISO date", ErrFormat)
	}
	return app.Case{Input: in, Label: label}, nil
}
