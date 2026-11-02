// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
)

func TestRequestIDIsKeptOnlyWhenBounded(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name, sent, want string
	}{
		{"none", "", "GENERATED"},
		{"valid", "req-42_a.B", "req-42_a.B"},
		{"longest", strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{"too long", strings.Repeat("a", 65), "GENERATED"},
		{"space", "a b", "GENERATED"},
		{"newline", "a\nb", "GENERATED"},
		{"quote", `a"b`, "GENERATED"},
		{"non ascii", "é", "GENERATED"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, logs := options(t, true)
			var header http.Header
			if tt.sent != "" {
				header = http.Header{"X-Request-Id": {tt.sent}}
			}
			res := serve(t, httpapi.NewHandler(opts), http.MethodGet, "/healthz", header)
			if got := res.Header().Get("X-Request-ID"); got != tt.want {
				t.Errorf("X-Request-ID = %q, want %q", got, tt.want)
			}
			lines := logs.Lines(t)
			if len(lines) != 1 || lines[0]["request_id"] != tt.want {
				t.Errorf("log = %v, want one line with request_id %q", lines, tt.want)
			}
		})
	}
}

func TestRequestIDDefaultsToARandomText(t *testing.T) {
	t.Parallel()

	opts, _ := options(t, true)
	opts.NewID = nil
	h := httpapi.NewHandler(opts)
	a := serve(t, h, http.MethodGet, "/healthz", nil).Header().Get("X-Request-ID")
	b := serve(t, h, http.MethodGet, "/healthz", nil).Header().Get("X-Request-ID")
	if len(a) < 26 || a == b {
		t.Errorf("generated identifiers %q and %q, want two distinct random texts", a, b)
	}
}
