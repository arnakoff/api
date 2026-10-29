// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func TestFormatDate(t *testing.T) {
	t.Parallel()

	plus2 := time.FixedZone("UTC+2", 2*60*60)
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{"pads day and month", time.Date(2026, time.March, 4, 12, 0, 0, 0, time.UTC), "04/03/2026"},
		{"two-digit day and month", time.Date(2026, time.October, 28, 0, 0, 0, 0, time.UTC), "28/10/2026"},
		{"uses the time's own location", time.Date(2026, time.October, 7, 23, 30, 0, 0, time.UTC).In(plus2), "08/10/2026"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := domain.FormatDate(tt.in); got != tt.want {
				t.Errorf("FormatDate(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
