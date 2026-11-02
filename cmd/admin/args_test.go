// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"io"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want command
	}{
		{"migrate", []string{"migrate"}, command{action: actionMigrate}},
		{"create", []string{"key", "create", "-label", "Bot Discord"}, command{action: actionCreate, label: "Bot Discord"}},
		{"create for test", []string{"key", "create", "-env", "test", "-label", "x"}, command{action: actionCreate, label: "x", env: "test"}},
		{"list", []string{"key", "list"}, command{action: actionList}},
		{"revoke", []string{"key", "revoke", "-id", "42"}, command{action: actionRevoke, id: 42}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs(tt.args, io.Discard)
			if err != nil || got != tt.want {
				t.Errorf("parseArgs(%q) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestParseArgsRejectsMisuse(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		nil,
		{"unknown"},
		{"migrate", "now"},
		{"key"},
		{"key", "delete"},
		{"key", "create"},
		{"key", "create", "-label", "x", "extra"},
		{"key", "list", "-label", "x"},
		{"key", "revoke"},
		{"key", "revoke", "-id", "0"},
		{"key", "revoke", "-id", "abc"},
	} {
		if _, err := parseArgs(args, io.Discard); err == nil || errors.Is(err, flag.ErrHelp) {
			t.Errorf("parseArgs(%q) error = %v, want a usage error", args, err)
		}
	}
}

func TestParseArgsHelp(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"-h"}, {"key", "create", "-h"}} {
		if _, err := parseArgs(args, io.Discard); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("parseArgs(%q) error = %v, want flag.ErrHelp", args, err)
		}
	}
}
