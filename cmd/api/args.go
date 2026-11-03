// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"fmt"
)

// command is what the process was started to do.
type command int

const (
	// serveCommand serves the API: the process started without argument.
	serveCommand command = iota + 1

	// probeCommand asks /healthz of the API this configuration serves.
	probeCommand
)

// errUsage is a misuse of the arguments.
var errUsage = errors.New("usage: api [probe]")

// parseArgs reads the arguments after the program name.
func parseArgs(args []string) (command, error) {
	switch {
	case len(args) == 0:
		return serveCommand, nil
	case len(args) == 1 && args[0] == "probe":
		return probeCommand, nil
	}
	return 0, fmt.Errorf("%w: unexpected arguments %q", errUsage, args)
}
