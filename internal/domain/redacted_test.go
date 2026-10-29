// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"testing"

	"github.com/arnakoff/api/internal/domain"
)

// A zero value reaches the logger when a caller logs a variable before
// parsing into it: Redacted must not panic on it.
func TestRedactedZeroValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    interface{ Redacted() string }
		want string
	}{
		{"Email", domain.Email{}, "***@"},
		{"Phone", domain.Phone{}, "***"},
		{"IBAN", domain.IBAN{}, "***"},
	}
	for _, tt := range tests {
		if got := tt.v.Redacted(); got != tt.want {
			t.Errorf("%s{}.Redacted() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
