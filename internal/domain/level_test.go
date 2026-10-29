// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

func TestLevel_ordersFromInfoToAlert(t *testing.T) {
	t.Parallel()

	ascending := []domain.Level{domain.LevelInfo, domain.LevelAttention, domain.LevelAlert}
	for i := 1; i < len(ascending); i++ {
		if ascending[i-1] >= ascending[i] {
			t.Errorf("%v (%d) is not below %v (%d)",
				ascending[i-1], int(ascending[i-1]), ascending[i], int(ascending[i]))
		}
	}
}

func TestLevel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level domain.Level
		want  string
	}{
		{domain.LevelInfo, "info"},
		{domain.LevelAttention, "attention"},
		{domain.LevelAlert, "alerte"},
		{domain.Level(0), "Level(0)"},
		{domain.Level(42), "Level(42)"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", int(tt.level), got, tt.want)
		}
	}
}
