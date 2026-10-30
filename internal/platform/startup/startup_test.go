// SPDX-License-Identifier: AGPL-3.0-only

package startup_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/startup"
)

// validKey is 32 bytes in hex, the shortest key config.Load accepts.
const validKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

// env returns a lookup function reading vars, like os.LookupEnv.
func env(vars map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}
}

// restoreDefault puts back the default logger Run replaces. Tests calling
// it change global state and must not run in parallel.
func restoreDefault(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
}

// onlyLine decodes buf, which must hold exactly one JSON line.
func onlyLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	out := strings.TrimSpace(buf.String())
	if strings.Count(out, "\n") != 0 || out == "" {
		t.Fatalf("got output %q, want one JSON line", out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", out, err)
	}
	return m
}

func TestRunStopsOnInvalidConfiguration(t *testing.T) {
	restoreDefault(t)
	cases := []struct {
		name string
		vars map[string]string
	}{
		{"missing key", nil},
		{"short key", map[string]string{config.EnvLogKey: validKey[:62]}},
		{"not hexadecimal", map[string]string{config.EnvLogKey: strings.Repeat("zz", 32)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			called := false
			code := startup.Run("api", env(tc.vars), &buf, func(context.Context, config.Config) error { called = true; return nil })
			if code == 0 {
				t.Errorf("Run() = 0, want non-zero")
			}
			if called {
				t.Errorf("run was called with an invalid configuration")
			}
			assertConfigError(t, &buf, tc.vars[config.EnvLogKey])
		})
	}
}

// assertConfigError checks buf holds one error line for the api command
// naming the log key variable, without quoting key.
func assertConfigError(t *testing.T, buf *bytes.Buffer, key string) {
	t.Helper()
	if key != "" && strings.Contains(buf.String(), key) {
		t.Errorf("output quotes the key: %s", buf)
	}
	line := onlyLine(t, buf)
	if line["level"] != "ERROR" || line["command"] != "api" {
		t.Errorf("line = %v, want level ERROR and command api", line)
	}
	if msg, _ := line["error"].(string); !strings.Contains(msg, config.EnvLogKey) {
		t.Errorf("error %q does not name %s", msg, config.EnvLogKey)
	}
}

func TestRunInstallsRedactingDefaultLogger(t *testing.T) {
	restoreDefault(t)
	var buf bytes.Buffer
	var got config.Config
	var ctxErr error
	code := startup.Run("api", env(map[string]string{
		config.EnvLogKey:   validKey,
		config.EnvLogLevel: "warn",
	}), &buf, func(ctx context.Context, cfg config.Config) error {
		got = cfg
		ctxErr = ctx.Err()
		slog.Info("dropped below warn")
		slog.Warn("written", "email", redactable{})
		return nil
	})
	if code != 0 {
		t.Fatalf("Run() = %d, want 0; output %s", code, &buf)
	}
	if ctxErr != nil {
		t.Errorf("run got a done context: %v", ctxErr)
	}
	if got.LogLevel != slog.LevelWarn {
		t.Errorf("run got LogLevel %v, want WARN", got.LogLevel)
	}
	line := onlyLine(t, &buf)
	if line["msg"] != "written" {
		t.Errorf("msg = %v, want written", line["msg"])
	}
	if strings.Contains(buf.String(), redactable{}.String()) {
		t.Errorf("default logger wrote the raw value: %s", &buf)
	}
}

func TestRunLogsRunError(t *testing.T) {
	restoreDefault(t)
	var buf bytes.Buffer
	code := startup.Run("eval", env(map[string]string{config.EnvLogKey: validKey}), &buf,
		func(context.Context, config.Config) error { return errors.New("corpus missing") })
	if code == 0 {
		t.Errorf("Run() = 0, want non-zero")
	}
	line := onlyLine(t, &buf)
	if line["level"] != "ERROR" || line["command"] != "eval" || line["error"] != "corpus missing" {
		t.Errorf("line = %v, want an ERROR line for eval with the run error", line)
	}
}

// redactable is a value with both a Redacted and a String method, which
// the redacting logger masks.
type redactable struct{}

func (redactable) String() string   { return "jean.dupont@exemple.fr" }
func (redactable) Redacted() string { return "***@exemple.fr" }
