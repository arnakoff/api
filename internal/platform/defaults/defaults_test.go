// SPDX-License-Identifier: AGPL-3.0-only

package defaults_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/defaults"
)

func TestSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		field   time.Duration
		want    time.Duration
		wantErr bool
	}{
		{"zero takes default", 0, time.Minute, false},
		{"positive kept", time.Second, time.Second, false},
		{"negative rejected", -time.Second, -time.Second, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			field := tt.field
			err := defaults.Set("timeout", &field, time.Minute)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Set err = %v, want error %t", err, tt.wantErr)
			}
			if field != tt.want {
				t.Errorf("field = %v, want %v", field, tt.want)
			}
		})
	}
}

func TestSetNamesTheField(t *testing.T) {
	t.Parallel()
	n := -1
	err := defaults.Set("attempts", &n, 3)
	if want := "negative attempts -1"; err == nil || err.Error() != want {
		t.Errorf("Set err = %v, want %q", err, want)
	}
}
