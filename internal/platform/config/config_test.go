// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

// validKey is 32 bytes in hex, the shortest key Load accepts.
const validKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

// env returns a lookup function reading vars, like os.LookupEnv.
func env(vars map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != config.DefaultAddr {
		t.Errorf("Addr = %q, want %q", cfg.Addr, config.DefaultAddr)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.DataDir != config.DefaultDataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, config.DefaultDataDir)
	}
	if got := len(cfg.LogKey.Bytes()); got != 32 {
		t.Errorf("LogKey is %d bytes, want 32", got)
	}
	if cfg.SourceACPR {
		t.Error("SourceACPR = true, want off until its licence is obtained")
	}
}

func TestLoadReadsEveryVariable(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_ADDR":        "127.0.0.1:9000",
		"ARNAKOFF_LOG_LEVEL":   "debug",
		"ARNAKOFF_LOG_KEY":     strings.ToUpper(validKey) + "20",
		"ARNAKOFF_DATA_DIR":    "/data/",
		"ARNAKOFF_SOURCE_ACPR": "true",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9000" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want DEBUG", cfg.LogLevel)
	}
	if got := len(cfg.LogKey.Bytes()); got != 33 {
		t.Errorf("LogKey is %d bytes, want 33", got)
	}
	if want := filepath.Clean("/data"); cfg.DataDir != want {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, want)
	}
	if !cfg.SourceACPR {
		t.Error("SourceACPR = false, want true")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, variable, value string
	}{
		{"addr without port", "ARNAKOFF_ADDR", "localhost"},
		{"addr with port zero", "ARNAKOFF_ADDR", ":0"},
		{"addr with port out of range", "ARNAKOFF_ADDR", ":65536"},
		{"addr with named port", "ARNAKOFF_ADDR", ":http"},
		{"empty addr", "ARNAKOFF_ADDR", ""},
		{"unknown level", "ARNAKOFF_LOG_LEVEL", "verbose"},
		{"empty level", "ARNAKOFF_LOG_LEVEL", ""},
		{"key not hex", "ARNAKOFF_LOG_KEY", strings.Repeat("zz", 32)},
		{"key too short", "ARNAKOFF_LOG_KEY", validKey[:62]},
		{"empty data dir", "ARNAKOFF_DATA_DIR", ""},
		{"source switch not a boolean", "ARNAKOFF_SOURCE_ACPR", "oui"},
		{"empty source switch", "ARNAKOFF_SOURCE_ACPR", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"ARNAKOFF_LOG_KEY": validKey, tc.variable: tc.value}
			_, err := config.Load(env(vars))
			if !errors.Is(err, config.ErrInvalid) {
				t.Fatalf("Load error = %v, want ErrInvalid", err)
			}
			if !strings.Contains(err.Error(), tc.variable) {
				t.Errorf("error %q does not name %s", err, tc.variable)
			}
		})
	}
}

func TestLoadRequiresTheLogKey(t *testing.T) {
	t.Parallel()
	_, err := config.Load(env(nil))
	if !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), "ARNAKOFF_LOG_KEY") {
		t.Fatalf("Load error = %v, want ErrInvalid naming ARNAKOFF_LOG_KEY", err)
	}
}

func TestLoadReportsEveryInvalidVariable(t *testing.T) {
	t.Parallel()
	_, err := config.Load(env(map[string]string{
		"ARNAKOFF_ADDR":      "nope",
		"ARNAKOFF_LOG_LEVEL": "nope",
	}))
	for _, name := range []string{"ARNAKOFF_ADDR", "ARNAKOFF_LOG_LEVEL", "ARNAKOFF_LOG_KEY"} {
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("error %v does not name %s", err, name)
		}
	}
}

func TestLoadErrorNeverQuotesTheKey(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"g" + validKey[1:], validKey[:40]} {
		_, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": key}))
		if err == nil {
			t.Fatalf("Load(%q) succeeded", key)
		}
		if strings.Contains(err.Error(), key[:8]) || strings.Contains(err.Error(), "'g'") {
			t.Errorf("error %q quotes the key", err)
		}
	}
}

func TestWritablePathsStayUnderTheDataDir(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY":  validKey,
		"ARNAKOFF_DATA_DIR": "/data",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rel, err := filepath.Rel(cfg.DataDir, cfg.SnapshotDir())
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		t.Fatalf("SnapshotDir %q is not inside DataDir %q", cfg.SnapshotDir(), cfg.DataDir)
	}
}
