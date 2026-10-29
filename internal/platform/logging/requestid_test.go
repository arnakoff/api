// SPDX-License-Identifier: AGPL-3.0-only

package logging_test

import (
	"context"
	"testing"

	"github.com/arnakoff/api/internal/platform/logging"
)

func TestRequestIDRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := logging.WithRequestID(context.Background(), "req-1")

	if got, ok := logging.RequestID(ctx); !ok || got != "req-1" {
		t.Errorf("RequestID() = %q, %v, want req-1, true", got, ok)
	}
	if got, ok := logging.RequestID(context.Background()); ok {
		t.Errorf("RequestID(empty context) = %q, true, want false", got)
	}
	if got, ok := logging.RequestID(logging.WithRequestID(context.Background(), "")); ok {
		t.Errorf("RequestID(empty id) = %q, true, want false", got)
	}
}

func TestLoggerAddsRequestID(t *testing.T) {
	t.Parallel()
	logger, buf := newTestLogger(t, logging.Config{})
	ctx := logging.WithRequestID(context.Background(), "req-1")

	logger.InfoContext(ctx, "with id")
	logger.With("source", "amf").InfoContext(ctx, "with attrs")
	logger.InfoContext(context.Background(), "without id")

	got := lines(t, buf)
	if len(got) != 3 {
		t.Fatalf("got %d log lines, want 3: %s", len(got), buf)
	}
	for i, want := range []any{"req-1", "req-1", nil} {
		if got[i]["request_id"] != want {
			t.Errorf("line %d request_id = %v, want %v", i, got[i]["request_id"], want)
		}
	}
	if got[1]["source"] != "amf" {
		t.Errorf("line 1 source = %v, want amf", got[1]["source"])
	}
}

func TestLoggerKeepsRequestIDAtTopLevel(t *testing.T) {
	t.Parallel()
	logger, buf := newTestLogger(t, logging.Config{})
	ctx := logging.WithRequestID(context.Background(), "req-1")

	logger.WithGroup("sync").InfoContext(ctx, "grouped", "source", "amf")

	got := only(t, buf)
	if got["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want req-1 at top level: %v", got["request_id"], got)
	}
	if field(t, got, "sync")["source"] != "amf" {
		t.Errorf("sync.source missing: %v", got)
	}
}
