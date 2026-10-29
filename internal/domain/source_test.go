// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

func TestSourceStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status domain.SourceStatus
		want   string
	}{
		{domain.SourceOK, "ok"},
		{domain.SourceUnavailable, "indisponible"},
		{domain.SourceNotApplicable, "non applicable"},
		{domain.SourceStatus(0), "SourceStatus(0)"},
		{domain.SourceStatus(9), "SourceStatus(9)"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("SourceStatus(%d).String() = %q, want %q", int(tt.status), got, tt.want)
		}
	}
}
