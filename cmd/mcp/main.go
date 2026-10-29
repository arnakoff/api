// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"os"

	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/startup"
)

func main() {
	os.Exit(startup.Run("mcp", os.LookupEnv, os.Stderr, run))
}

// run is the work of the command, once its configuration is loaded and the
// redacting logger is the slog default.
func run(config.Config) error {
	return nil
}
