// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
)

func TestProbes(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		ready  bool
		target string
		status int
		body   string
	}{
		{"alive", false, "/healthz", http.StatusOK, "ok"},
		{"ready", true, "/readyz", http.StatusOK, "ok"},
		{"not ready", false, "/readyz", http.StatusServiceUnavailable, "indisponible"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, _ := options(t, tt.ready)
			res := serve(t, httpapi.NewHandler(opts), http.MethodGet, tt.target, nil)
			if res.Code != tt.status {
				t.Errorf("status = %d, want %d", res.Code, tt.status)
			}
			if ct := res.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			if got := decode(t, res)["status"]; got != tt.body {
				t.Errorf("status field = %v, want %q", got, tt.body)
			}
		})
	}
}
