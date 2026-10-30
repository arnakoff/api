// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"io"
	"reflect"
	"testing"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want options
	}{
		{"online", nil, options{corpus: defaultCorpus}},
		{"offline", []string{"-offline"}, options{corpus: defaultCorpus, offline: true}},
		{"other corpus", []string{"-corpus", "x"}, options{corpus: "x"}},
		{
			"without the blacklists",
			[]string{"-offline", "-without", "amf,ACPR"},
			options{corpus: defaultCorpus, offline: true, without: []string{"amf", "acpr"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseFlags(tt.args, io.Discard)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFlags(%q) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestParseFlagsRejectsMisuse(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"-without", "opensanctions"},
		{"-without", ""},
		{"-without", "amf,,acpr"},
		{"-corpus", ""},
		{"extra"},
		{"-unknown"},
	} {
		if _, err := parseFlags(args, io.Discard); err == nil {
			t.Errorf("parseFlags(%q): want an error", args)
		}
	}
}

func TestParseFlagsHelp(t *testing.T) {
	t.Parallel()
	if _, err := parseFlags([]string{"-h"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("parseFlags(-h) error = %v, want flag.ErrHelp", err)
	}
}

func TestLeftOut(t *testing.T) {
	t.Parallel()
	opts := options{without: []string{"rdap", "amf"}}
	if !opts.leftOut("RDAP") || opts.leftOut("SIRENE") {
		t.Errorf("leftOut: want RDAP left out, SIRENE kept")
	}
}
