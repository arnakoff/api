// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"strconv"

	"github.com/arnakoff/api/internal/platform/logging"
)

// Environment variables read by Load.
const (
	EnvAddr     = "ARNAKOFF_ADDR"
	EnvLogLevel = "ARNAKOFF_LOG_LEVEL"
	EnvLogKey   = "ARNAKOFF_LOG_KEY"
	EnvDataDir  = "ARNAKOFF_DATA_DIR"

	EnvSourceACPR = "ARNAKOFF_SOURCE_ACPR"
)

// Defaults applied by Load to an unset variable. ARNAKOFF_LOG_KEY has none.
const (
	DefaultAddr     = ":8100"
	DefaultLogLevel = "info"
	DefaultDataDir  = "var"

	// DefaultSourceACPR keeps the ACPR blacklist off: its reuse licence is
	// still to be obtained (docs/sources.md).
	DefaultSourceACPR = "false"
)

// MinLogKeyBytes is the shortest log key Load accepts, once hex-decoded:
// the minimum of logging.New.
const MinLogKeyBytes = logging.MinKeyBytes

// ErrInvalid means a variable holds a value Load cannot use.
var ErrInvalid = errors.New("invalid configuration")

// Config is the configuration of every command.
type Config struct {
	// Addr is the TCP address the HTTP API listens on, "host:port" with a
	// numeric port; an empty host means every interface.
	Addr string

	// LogLevel is the minimum level written to the log.
	LogLevel slog.Level

	// LogKey is the HMAC key that pseudonymises personal data in the log.
	LogKey Secret

	// DataDir is the only directory the commands write to: the data
	// volume. Every other path may be read-only.
	DataDir string

	// SourceACPR turns the ACPR blacklist on, once its reuse licence is
	// obtained.
	SourceACPR bool
}

// SnapshotDir is where the validated snapshots of the sources are stored,
// inside DataDir.
func (c Config) SnapshotDir() string {
	return filepath.Join(c.DataDir, "snapshots")
}

// Load reads the configuration with lookup, such as os.LookupEnv. A
// variable that is set, even to an empty string, is used as is; an unset
// one takes its default. It returns every invalid variable at once, each
// wrapping ErrInvalid; an error never quotes the value of a Secret.
func Load(lookup func(string) (string, bool)) (Config, error) {
	var cfg Config
	errs := []error{
		parse(lookup, EnvAddr, DefaultAddr, &cfg.Addr, parseAddr),
		parse(lookup, EnvLogLevel, DefaultLogLevel, &cfg.LogLevel, parseLevel),
		parse(lookup, EnvLogKey, "", &cfg.LogKey, parseKey),
		parse(lookup, EnvDataDir, DefaultDataDir, &cfg.DataDir, parseDir),
		parse(lookup, EnvSourceACPR, DefaultSourceACPR, &cfg.SourceACPR, strconv.ParseBool),
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// parse sets *field to the parsed value of the variable name, or of def
// when it is unset.
func parse[T any](lookup func(string) (string, bool), name, def string, field *T, parseValue func(string) (T, error)) error {
	raw, ok := lookup(name)
	if !ok {
		raw = def
	}
	v, err := parseValue(raw)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInvalid, name, err)
	}
	*field = v
	return nil
}

// parseAddr accepts "host:port" with a port from 1 to 65535.
func parseAddr(s string) (string, error) {
	_, port, err := net.SplitHostPort(s)
	if err != nil {
		return "", err
	}
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil || n == 0 {
		return "", fmt.Errorf("port %q is not a number from 1 to 65535", port)
	}
	return s, nil
}

// parseLevel accepts a slog level name, such as "debug" or "WARN".
func parseLevel(s string) (slog.Level, error) {
	var l slog.Level
	err := l.UnmarshalText([]byte(s))
	return l, err
}

// parseKey accepts at least MinLogKeyBytes in hex. Its errors never quote
// the input: the decoder's would show the offending character.
func parseKey(s string) (Secret, error) {
	if s == "" {
		return nil, errors.New("required, generate one with: openssl rand -hex 32")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, errors.New("not hexadecimal")
	}
	if len(b) < MinLogKeyBytes {
		return nil, fmt.Errorf("%d bytes once decoded, want at least %d", len(b), MinLogKeyBytes)
	}
	return Secret(b), nil
}

// parseDir accepts a non-empty path and cleans it.
func parseDir(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty path")
	}
	return filepath.Clean(s), nil
}
