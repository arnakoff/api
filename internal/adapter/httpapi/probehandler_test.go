// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
)

func TestProbeHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ready  bool
		method string
		path   string
		want   int
	}{
		{"alive", false, http.MethodGet, "/healthz", http.StatusOK},
		{"not ready", false, http.MethodGet, "/readyz", http.StatusServiceUnavailable},
		{"ready", true, http.MethodGet, "/readyz", http.StatusOK},
		{"only the probes", true, http.MethodGet, "/v1/check", http.StatusNotFound},
		{"read only", true, http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			httpapi.NewProbeHandler(readiness(tt.ready)).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, nil))
			if rec.Code != tt.want {
				t.Errorf("%s %s = %d, want %d", tt.method, tt.path, rec.Code, tt.want)
			}
		})
	}
}
