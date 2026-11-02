// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/domain"
)

// emptyCheck answers an empty report at once: the benchmark measures the
// handler around the check.
type emptyCheck struct{}

func (emptyCheck) Check(context.Context, domain.Input) domain.Report { return domain.Report{} }

func (c emptyCheck) CheckAll(ctx context.Context, ins []domain.Input) []domain.Report {
	return checkEach(ctx, c.Check, ins)
}

func BenchmarkCheckRoute(b *testing.B) {
	h := httpapi.NewHandler(httpapi.Options{
		Ready: readiness(true), Check: emptyCheck{}, Auth: keyring{},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), NewID: func() string { return "BENCH" },
		ProblemBase: problemBase, BatchMaxItems: 50,
		KeyLimit: frozenLimiter(1 << 30), IPLimit: frozenLimiter(1 << 30),
	})
	req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/v1/check?q=armen.ltd", nil)
	req.Header.Set("Authorization", "Bearer "+testKey)
	for b.Loop() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}
