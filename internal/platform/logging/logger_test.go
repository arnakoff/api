// SPDX-License-Identifier: AGPL-3.0-only

package logging_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/logging"
)

// testKey is a 32-byte HMAC key for tests only.
var testKey = []byte("0123456789abcdef0123456789abcdef")

// newTestLogger returns a logger writing to the returned buffer.
func newTestLogger(t *testing.T, cfg logging.Config) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	if cfg.Key == nil {
		cfg.Key = testKey
	}
	var buf bytes.Buffer
	logger, err := logging.New(&buf, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return logger, &buf
}

// lines decodes every JSON line of buf.
func lines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("json.Unmarshal(%q) error = %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// only decodes buf, which must hold exactly one JSON line.
func only(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	got := lines(t, buf)
	if len(got) != 1 {
		t.Fatalf("got %d log lines, want 1: %s", len(got), buf)
	}
	return got[0]
}

func TestNewRejectsShortKey(t *testing.T) {
	t.Parallel()

	for _, key := range [][]byte{nil, {}, testKey[:logging.MinKeyBytes-1]} {
		_, err := logging.New(&bytes.Buffer{}, logging.Config{Key: key})
		if !errors.Is(err, logging.ErrInvalidConfig) {
			t.Errorf("New(key of %d bytes) error = %v, want ErrInvalidConfig", len(key), err)
		}
	}
}

func TestLoggerWritesJSON(t *testing.T) {
	t.Parallel()
	logger, buf := newTestLogger(t, logging.Config{})

	logger.Info("snapshot loaded", slog.Int("entries", 42))

	got := only(t, buf)
	if got["level"] != "INFO" || got["msg"] != "snapshot loaded" || got["entries"] != 42.0 {
		t.Errorf("log line = %v, want level INFO, msg and entries 42", got)
	}
}

func TestLoggerHonoursLevel(t *testing.T) {
	t.Parallel()
	logger, buf := newTestLogger(t, logging.Config{Level: slog.LevelWarn})

	logger.Info("dropped")
	logger.Warn("kept")

	got := only(t, buf)
	if got["msg"] != "kept" {
		t.Errorf("msg = %v, want kept", got["msg"])
	}
}
