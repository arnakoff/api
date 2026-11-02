// SPDX-License-Identifier: AGPL-3.0-only

// Package loggingtest builds redacting loggers for tests, writing to a
// buffer the test reads back.
package loggingtest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"

	"github.com/arnakoff/api/internal/platform/logging"
)

// Buffer collects log output, safely from several goroutines.
type Buffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends p.
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns everything written.
func (b *Buffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Lines decodes every JSON line written, in order. A line that is not JSON
// fails the test.
func (b *Buffer) Lines(t testing.TB) []map[string]any {
	t.Helper()
	var out []map[string]any
	sc := bufio.NewScanner(bytes.NewReader([]byte(b.String())))
	for sc.Scan() {
		var line map[string]any
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			t.Fatalf("log line %q: %v", sc.Text(), err)
		}
		out = append(out, line)
	}
	return out
}

// New returns a redacting logger, as logging.New makes it, keyed with a
// dummy key and writing to the returned buffer.
func New(t testing.TB) (*slog.Logger, *Buffer) {
	t.Helper()
	buf := &Buffer{}
	logger, err := logging.New(buf, logging.Config{Key: bytes.Repeat([]byte{0x42}, logging.MinKeyBytes)})
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}
	return logger, buf
}
