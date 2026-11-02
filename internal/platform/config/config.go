// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arnakoff/api/internal/platform/logging"
)

// Environment variables read by Load.
const (
	EnvAddr     = "ARNAKOFF_ADDR"
	EnvLogLevel = "ARNAKOFF_LOG_LEVEL"
	EnvLogKey   = "ARNAKOFF_LOG_KEY"
	EnvDataDir  = "ARNAKOFF_DATA_DIR"

	EnvSourceACPR = "ARNAKOFF_SOURCE_ACPR"

	EnvAlertWebhook = "ARNAKOFF_ALERT_WEBHOOK"

	EnvProblemBaseURL = "ARNAKOFF_PROBLEM_BASE_URL"

	EnvBatchMaxItems = "ARNAKOFF_BATCH_MAX_ITEMS"

	EnvDatabaseURL  = "ARNAKOFF_DATABASE_URL"
	EnvAPIKeySecret = "ARNAKOFF_API_KEY_SECRET" //nolint:gosec // G101: the name of a variable, not its value.
	EnvAPIKeyEnv    = "ARNAKOFF_API_KEY_ENV"    //nolint:gosec // G101: the name of a variable, not its value.
)

// Defaults applied by Load to an unset variable. ARNAKOFF_LOG_KEY has none.
const (
	DefaultAddr     = ":8100"
	DefaultLogLevel = "info"
	DefaultDataDir  = "var"

	// DefaultSourceACPR keeps the ACPR blacklist off: its reuse licence is
	// still to be obtained (docs/sources.md).
	DefaultSourceACPR = "false"

	// DefaultProblemBaseURL is where arnakoff/web documents the problems
	// the API answers, one page per code.
	DefaultProblemBaseURL = "https://arnakoff.com/docs/erreurs/"
)

// Bounds of ARNAKOFF_BATCH_MAX_ITEMS.
const (
	// DefaultBatchMaxItems is the most items POST /v1/check/batch accepts,
	// as the owner set it; the contract's example states it.
	DefaultBatchMaxItems = 50

	// MaxBatchMaxItems is the highest value Load accepts: beyond, one
	// request would weigh like a thousand checks.
	MaxBatchMaxItems = 1000
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

	// AlertWebhook is the HTTPS URL a failed or rejected sync is posted
	// to; nil when unset or empty, and then alerts are only logged. It is a
	// Secret: a webhook URL usually carries its credential.
	AlertWebhook Secret

	// ProblemBaseURL is the HTTPS URL of the documentation of the API's
	// problems, ending with a slash: the type of a problem is its code
	// under it.
	ProblemBaseURL string

	// BatchMaxItems is the most items POST /v1/check/batch accepts, from 1
	// to MaxBatchMaxItems.
	BatchMaxItems int

	// DatabaseURL is the PostgreSQL connection string of the API key store;
	// nil when unset or empty. RequireKeyStore tells whether it is set.
	DatabaseURL Secret

	// APIKeySecret is the HMAC key the API keys are digested with, kept
	// out of the database; nil when unset or empty.
	APIKeySecret Secret

	// APIKeyEnv is the environment of the API keys accepted: "live" or
	// "test".
	APIKeyEnv string
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
		parse(lookup, EnvAlertWebhook, "", &cfg.AlertWebhook, parseWebhook),
		parse(lookup, EnvProblemBaseURL, DefaultProblemBaseURL, &cfg.ProblemBaseURL, parseProblemBase),
		parse(lookup, EnvBatchMaxItems, strconv.Itoa(DefaultBatchMaxItems), &cfg.BatchMaxItems, parseBatchMaxItems),
		parse(lookup, EnvDatabaseURL, "", &cfg.DatabaseURL, parseOptionalSecret),
		parse(lookup, EnvAPIKeySecret, "", &cfg.APIKeySecret, parseOptionalKey),
		parse(lookup, EnvAPIKeyEnv, DefaultAPIKeyEnv, &cfg.APIKeyEnv, parseAPIKeyEnv),
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

// parseWebhook accepts an empty string, as no webhook, or an absolute HTTPS
// URL with a host and no credentials. Its errors never quote the input.
func parseWebhook(s string) (Secret, error) {
	if s == "" {
		return nil, nil
	}
	if _, err := parseHTTPS(s); err != nil {
		return nil, err
	}
	return Secret(s), nil
}

// parseProblemBase accepts an absolute HTTPS URL with a host, no
// credentials, no query and no fragment, whose path ends with a slash: a
// code appended to it is the URL of the code's page.
func parseProblemBase(s string) (string, error) {
	u, err := parseHTTPS(s)
	switch {
	case err != nil:
		return "", err
	case u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(s, "#"):
		return "", errors.New("a query or a fragment is not allowed")
	case !strings.HasSuffix(u.Path, "/"):
		return "", errors.New("the path must end with a slash")
	}
	return s, nil
}

// parseHTTPS parses an absolute HTTPS URL with a host and no credentials.
// Its errors never quote the input.
func parseHTTPS(s string) (*url.URL, error) {
	u, err := url.Parse(s)
	switch {
	case err != nil:
		return nil, errors.New("not a URL")
	case u.Scheme != "https" || u.Host == "":
		return nil, errors.New("not an https:// URL with a host")
	case u.User != nil:
		return nil, errors.New("credentials before the host are not allowed")
	}
	return u, nil
}

// parseBatchMaxItems accepts a decimal number from 1 to MaxBatchMaxItems.
func parseBatchMaxItems(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > MaxBatchMaxItems {
		return 0, fmt.Errorf("%q is not a number from 1 to %d", s, MaxBatchMaxItems)
	}
	return n, nil
}

// parseDir accepts a non-empty path and cleans it.
func parseDir(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty path")
	}
	return filepath.Clean(s), nil
}
