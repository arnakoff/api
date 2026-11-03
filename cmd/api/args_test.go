// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		args []string
		want command
		err  error
	}{
		{"no argument serves", nil, serveCommand, nil},
		{"probe", []string{"probe"}, probeCommand, nil},
		{"unknown command", []string{"serve"}, 0, errUsage},
		{"extra argument", []string{"probe", "now"}, 0, errUsage},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs(tt.args)
			if !errors.Is(err, tt.err) || got != tt.want {
				t.Errorf("parseArgs(%q) = %v, %v; want %v, %v", tt.args, got, err, tt.want, tt.err)
			}
		})
	}
}
