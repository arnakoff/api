// SPDX-License-Identifier: AGPL-3.0-only

package startup

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/logging"
)

// Process exit codes returned by Run.
const (
	exitOK      = 0
	exitFailure = 1
)

// Run sets the command name up and runs it: it loads the configuration with
// lookup, such as os.LookupEnv, makes a redacting logger writing JSON lines
// to w the slog default and calls run. It returns the process exit code: 0
// when run succeeds, 1 after logging the error otherwise. run is not called
// when the configuration is invalid.
func Run(name string, lookup func(string) (string, bool), w io.Writer, run func(config.Config) error) int {
	cfg, logger, err := setup(lookup, w)
	if err != nil {
		// No key to mask with: configuration errors never quote a
		// secret, and the environment holds no personal data.
		fail(slog.New(slog.NewJSONHandler(w, nil)), name, "cannot start", err)
		return exitFailure
	}
	slog.SetDefault(logger)
	if err := run(cfg); err != nil {
		fail(logger, name, "command failed", err)
		return exitFailure
	}
	return exitOK
}

// setup loads the configuration and builds the redacting logger from it.
func setup(lookup func(string) (string, bool), w io.Writer) (config.Config, *slog.Logger, error) {
	cfg, err := config.Load(lookup)
	if err != nil {
		return config.Config{}, nil, err
	}
	logger, err := logging.New(w, logging.Config{Level: cfg.LogLevel, Key: cfg.LogKey.Bytes()})
	if err != nil {
		return config.Config{}, nil, fmt.Errorf("logger: %w", err)
	}
	return cfg, logger, nil
}

// fail writes err as one error line naming the command.
func fail(logger *slog.Logger, name, msg string, err error) {
	logger.Error(msg, "command", name, "error", err.Error())
}
