// SPDX-License-Identifier: AGPL-3.0-only

package loggingtest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/logging"
	"github.com/arnakoff/api/internal/platform/logging/loggingtest"
)

func TestNewCollectsTheLinesOfARedactingLogger(t *testing.T) {
	t.Parallel()

	logger, logs := loggingtest.New(t)
	ctx := logging.WithRequestID(context.Background(), "req-1")
	logger.InfoContext(ctx, "first", "n", 1)
	logger.WarnContext(ctx, "second")

	lines := logs.Lines(t)
	if len(lines) != 2 || lines[0]["msg"] != "first" || lines[1]["level"] != "WARN" {
		t.Fatalf("Lines() = %v, want the two lines in order", lines)
	}
	if lines[0]["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want the context's", lines[0]["request_id"])
	}
	if !strings.Contains(logs.String(), `"msg":"second"`) {
		t.Errorf("String() = %q, want the raw output", logs.String())
	}
}
