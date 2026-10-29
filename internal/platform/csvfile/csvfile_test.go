// SPDX-License-Identifier: AGPL-3.0-only

package csvfile_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/csvfile"
)

var errFormat = errors.New("unexpected test file format")

var header = []string{"code", "name"}

// collect returns the records Each hands over, copied.
func collect(t *testing.T, in string) ([][]string, error) {
	t.Helper()
	r, err := csvfile.Open(strings.NewReader(in), header, errFormat)
	if err != nil {
		return nil, err
	}
	var got [][]string
	err = r.Each(func(rec []string) error {
		got = append(got, slices.Clone(rec))
		return nil
	})
	return got, err
}

func TestEach(t *testing.T) {
	t.Parallel()
	got, err := collect(t, "code;name\r\nFRTE;Orange\r\nSFR0;\"Société; française\"\n")
	if err != nil {
		t.Fatalf("Each() error = %v", err)
	}
	want := [][]string{{"FRTE", "Orange"}, {"SFR0", "Société; française"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("Each() records = %q, want %q", got, want)
	}
}

func TestOpenErrors(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"empty":           "",
		"other header":    "code;nom\n",
		"comma separated": "code,name\n",
		"longer header":   "code;name;date\n",
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := collect(t, in); !errors.Is(err, errFormat) {
				t.Errorf("Open() error = %v, want %v", err, errFormat)
			}
		})
	}
}

func acceptAll([]string) error { return nil }

func TestEachErrors(t *testing.T) {
	t.Parallel()
	errRow := errors.New("bad row")
	refuseSFR := func(rec []string) error {
		if rec[0] == "SFR0" {
			return errRow
		}
		return nil
	}
	tests := []struct {
		name string
		in   string
		add  func([]string) error
		want error
		line string
	}{
		{"missing field", "code;name\nFRTE;Orange\nSFR0\n", acceptAll, errFormat, "line 3"},
		{"bare quote", "code;name\nFRTE;Or\"ange\n", acceptAll, errFormat, "line 2"},
		{"row refused", "code;name\nFRTE;Orange\nSFR0;SFR\n", refuseSFR, errRow, "line 3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, err := csvfile.Open(strings.NewReader(tt.in), header, errFormat)
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}
			err = r.Each(tt.add)
			if !errors.Is(err, tt.want) || !strings.Contains(err.Error(), tt.line) {
				t.Errorf("Each() error = %v, want %v at %s", err, tt.want, tt.line)
			}
		})
	}
}

func TestErrorsLeaveValuesOut(t *testing.T) {
	t.Parallel()
	_, err := collect(t, "contact@exemple.fr;name\n")
	if !errors.Is(err, errFormat) || strings.Contains(err.Error(), "exemple") {
		t.Errorf("Open() error = %v, want errFormat without the value", err)
	}
}
