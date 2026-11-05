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
		{"migrate listening", []string{"migrate", "-listen", ":8080"}, command{action: actionMigrate, listen: ":8080"}},
		{"create", []string{"key", "create", "-label", "Bot Discord"}, command{action: actionCreate, label: "Bot Discord"}},
		{"create for test", []string{"key", "create", "-env", "test", "-label", "x"}, command{action: actionCreate, label: "x", env: "test"}},
		{"list", []string{"key", "list"}, command{action: actionList}},
		{"backup", []string{"backup"}, command{action: actionBackup}},
		{"backup listening", []string{"backup", "-listen", ":8080"}, command{action: actionBackup, listen: ":8080"}},
		{"restore", []string{"restore", "-file", "keys.tar"}, command{action: actionRestore, file: "keys.tar"}},
		{"restore listening", []string{"restore", "-listen", ":8080"}, command{action: actionRestore, listen: ":8080"}},
		{"revoke", []string{"key", "revoke", "-id", "42"}, command{action: actionRevoke, id: 42}},
		{"erase", []string{"account", "erase"}, command{action: actionErase}},
		{"erase listening", []string{"account", "erase", "-listen", ":8080"}, command{action: actionErase, listen: ":8080"}},
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
		{"key", "list", "-listen", ":8080"},
		{"key", "revoke"},
		{"key", "revoke", "-id", "0"},
		{"key", "revoke", "-id", "abc"},
		{"backup", "now"},
		{"backup", "-file", "keys.tar"},
		{"restore"},
		{"restore", "-file", "keys.tar", "-listen", ":8080"},
		{"account"},
		{"account", "delete"},
		{"account", "erase", "dev@exemple.fr"},
		{"account", "erase", "-email", "dev@exemple.fr"},
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
