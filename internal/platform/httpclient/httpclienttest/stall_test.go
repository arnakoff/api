// SPDX-License-Identifier: AGPL-3.0-only

package httpclienttest_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

func TestStallAbortsOnceTheRequestEnds(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	defer func() {
		if got, _ := recover().(error); !errors.Is(got, http.ErrAbortHandler) {
			t.Errorf("Stall() recovered %v, want http.ErrAbortHandler", got)
		}
		if rec.Flushed || rec.Body.Len() != 0 {
			t.Errorf("Stall() wrote a response")
		}
	}()
	httpclienttest.Stall(rec, req)
}
