// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"io"
	"testing"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want options
	}{
		{"scheduled", nil, options{}},
		{"once", []string{"-once"}, options{once: true}},
		{"dry run of one source", []string{"-once", "-dry-run", "-only", "AMF PSAN"}, options{once: true, dryRun: true, only: "AMF PSAN"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseFlags(tt.args, io.Discard)
			if err != nil || got != tt.want {
				t.Errorf("parseFlags(%q) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestParseFlagsRejectsMisuse(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"-dry-run"},
		{"-only", "AMF"},
		{"-once", "-only", ""},
		{"-unknown"},
		{"-once", "extra"},
	} {
		if _, err := parseFlags(args, io.Discard); err == nil || errors.Is(err, flag.ErrHelp) {
			t.Errorf("parseFlags(%q) = %v, want a usage error", args, err)
		}
	}
}

func TestParseFlagsHelp(t *testing.T) {
	t.Parallel()
	if _, err := parseFlags([]string{"-h"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("parseFlags(-h) = %v, want %v", err, flag.ErrHelp)
	}
}
