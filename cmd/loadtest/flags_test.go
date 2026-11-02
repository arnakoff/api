// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"io"
	"reflect"
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()
	defaults := options{corpus: defaultCorpus, pprof: defaultPprof}
	tests := []struct {
		name string
		args []string
		want options
	}{
		{"defaults", nil, defaults},
		{"latency", []string{"-latency", "150ms"}, options{corpus: defaultCorpus, pprof: defaultPprof, latency: 150 * time.Millisecond}},
		{"no profiling", []string{"-pprof", ""}, options{corpus: defaultCorpus}},
		{
			"keyed targets",
			[]string{"-targets", "http://127.0.0.1:8100", "-keys", "k"},
			options{corpus: defaultCorpus, pprof: defaultPprof, targets: "http://127.0.0.1:8100", keys: "k"},
		},
		{"other corpus", []string{"-corpus", "x"}, options{corpus: "x", pprof: defaultPprof}},
		{
			"targets",
			[]string{"-targets", "http://127.0.0.1:8100"},
			options{corpus: defaultCorpus, pprof: defaultPprof, targets: "http://127.0.0.1:8100"},
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

func TestParseFlagsRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{"an argument", []string{"extra"}},
		{"an empty corpus", []string{"-corpus", ""}},
		{"a negative latency", []string{"-latency", "-1s"}},
		{"profiling off the loopback", []string{"-pprof", "0.0.0.0:6060"}},
		{"profiling on every interface", []string{"-pprof", ":6060"}},
		{"profiling on a name", []string{"-pprof", "localhost:6060"}},
		{"targets off the loopback", []string{"-targets", "http://example.com"}},
		{"targets with a path", []string{"-targets", "http://127.0.0.1:8100/v1"}},
		{"keys without targets", []string{"-keys", "k"}},
		{"an unknown flag", []string{"-rate", "10"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseFlags(tt.args, io.Discard); err == nil {
				t.Errorf("parseFlags(%q): want an error", tt.args)
			}
		})
	}
}

func TestParseFlagsHelp(t *testing.T) {
	t.Parallel()

	_, err := parseFlags([]string{"-h"}, io.Discard)

	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("parseFlags(-h): err %v, want flag.ErrHelp", err)
	}
}
