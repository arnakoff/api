// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/arnakoff/api/internal/domain"
)

// zeros is a random source of zero bytes only: every character drawn from
// it is the first of the alphabet, '0'.
type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestNewAPIKeyFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		env    domain.KeyEnvironment
		prefix string
	}{
		{domain.KeyLive, "ark_live_"},
		{domain.KeyTest, "ark_test_"},
	}
	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			t.Parallel()
			k, err := domain.NewAPIKey(tt.env, zeros{})
			if err != nil {
				t.Fatalf("NewAPIKey() error = %v", err)
			}
			s := k.String()
			if !strings.HasPrefix(s, tt.prefix) || len(s) != len(tt.prefix)+36 {
				t.Fatalf("NewAPIKey() = %q, want %s and 36 characters", s, tt.prefix)
			}
			if got := k.Environment(); got != tt.env {
				t.Errorf("Environment() = %v, want %v", got, tt.env)
			}
			back, err := domain.ParseAPIKey(s)
			if err != nil || back != k {
				t.Errorf("ParseAPIKey(%q) = %v, %v, want the same key", s, back, err)
			}
		})
	}
}

func TestNewAPIKeyIsRandom(t *testing.T) {
	t.Parallel()

	a, errA := domain.NewAPIKey(domain.KeyLive, bytes.NewReader(bytes.Repeat([]byte{1}, 64)))
	b, errB := domain.NewAPIKey(domain.KeyLive, bytes.NewReader(bytes.Repeat([]byte{2}, 64)))
	if errA != nil || errB != nil {
		t.Fatalf("NewAPIKey() errors = %v, %v", errA, errB)
	}
	if a == b {
		t.Errorf("NewAPIKey() gave %q twice from different random bytes", a)
	}
}

// TestNewAPIKeyRejectsBiasedBytes checks the rejection sampling: a byte of
// 248 or more would make the first characters of the alphabet likelier, so
// it is skipped.
func TestNewAPIKeyRejectsBiasedBytes(t *testing.T) {
	t.Parallel()

	random := append(bytes.Repeat([]byte{255}, 10), bytes.Repeat([]byte{1}, 30)...)
	k, err := domain.NewAPIKey(domain.KeyLive, bytes.NewReader(random))
	if err != nil {
		t.Fatalf("NewAPIKey() error = %v", err)
	}
	if got := k.String()[len("ark_live_"):][:30]; got != strings.Repeat("1", 30) {
		t.Errorf("random part = %q, want thirty 1", got)
	}
}

func TestNewAPIKeyErrors(t *testing.T) {
	t.Parallel()

	if _, err := domain.NewAPIKey(domain.KeyLive, iotest.ErrReader(errors.New("boom"))); err == nil {
		t.Error("NewAPIKey() with a failing random source: no error")
	}
	if _, err := domain.NewAPIKey(domain.KeyEnvironment(0), zeros{}); !errors.Is(err, domain.ErrInvalidAPIKey) {
		t.Errorf("NewAPIKey() with no environment error = %v, want ErrInvalidAPIKey", err)
	}
}

func TestParseAPIKeyRejects(t *testing.T) {
	t.Parallel()

	k, err := domain.NewAPIKey(domain.KeyTest, zeros{})
	if err != nil {
		t.Fatal(err)
	}
	valid := k.String()
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"prefix only", "ark_test_"},
		{"unknown environment", "ark_prod_" + valid[len("ark_test_"):]},
		{"other environment", "ark_live_" + valid[len("ark_test_"):]},
		{"wrong checksum", valid[:len(valid)-1] + "1"},
		{"one character short", valid[:len(valid)-1]},
		{"one character long", valid + "0"},
		{"not base62", valid[:12] + "-" + valid[13:]},
		{"non-ASCII", valid[:12] + "é" + valid[14:]},
		{"surrounding space", " " + valid},
		{"upper-case prefix", strings.ToUpper(valid[:9]) + valid[9:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := domain.ParseAPIKey(tt.in); !errors.Is(err, domain.ErrInvalidAPIKey) {
				t.Errorf("ParseAPIKey(%q) error = %v, want ErrInvalidAPIKey", tt.in, err)
			}
		})
	}
}

func TestAPIKeyHintAndRedacted(t *testing.T) {
	t.Parallel()

	k, err := domain.NewAPIKey(domain.KeyLive, zeros{})
	if err != nil {
		t.Fatal(err)
	}
	s := k.String()
	if got, want := k.Hint(), s[len(s)-4:]; got != want {
		t.Errorf("Hint() = %q, want %q", got, want)
	}
	if got, want := k.Redacted(), "ark_live_…"+k.Hint(); got != want {
		t.Errorf("Redacted() = %q, want %q", got, want)
	}
}

func TestParseKeyEnvironment(t *testing.T) {
	t.Parallel()

	for _, env := range []domain.KeyEnvironment{domain.KeyLive, domain.KeyTest} {
		got, err := domain.ParseKeyEnvironment(env.String())
		if err != nil || got != env {
			t.Errorf("ParseKeyEnvironment(%q) = %v, %v, want %v", env.String(), got, err, env)
		}
	}
	for _, in := range []string{"", "LIVE", "prod"} {
		if _, err := domain.ParseKeyEnvironment(in); !errors.Is(err, domain.ErrInvalidAPIKey) {
			t.Errorf("ParseKeyEnvironment(%q) error = %v, want ErrInvalidAPIKey", in, err)
		}
	}
}
