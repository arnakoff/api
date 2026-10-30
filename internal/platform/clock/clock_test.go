// SPDX-License-Identifier: AGPL-3.0-only

package clock_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/clock"
)

func TestSystemNowIsTheCurrentTime(t *testing.T) {
	t.Parallel()
	before := time.Now()
	got := clock.System{}.Now()
	if got.Before(before) || got.After(time.Now()) {
		t.Errorf("Now() = %v, not between the calls around it", got)
	}
}

func TestSystemSleepWaits(t *testing.T) {
	t.Parallel()
	start := time.Now()
	if err := (clock.System{}).Sleep(t.Context(), 10*time.Millisecond); err != nil {
		t.Fatalf("Sleep() = %v", err)
	}
	if waited := time.Since(start); waited < 10*time.Millisecond {
		t.Errorf("Sleep() returned after %v, want at least 10ms", waited)
	}
}

func TestSystemSleepStopsWithTheContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := (clock.System{}).Sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Errorf("Sleep() = %v, want %v", err, context.Canceled)
	}
}

func TestJitterStaysUnderItsBound(t *testing.T) {
	t.Parallel()
	for _, bound := range []time.Duration{-time.Second, 0, 1, time.Minute} {
		for range 100 {
			if got := clock.Jitter(bound); got < 0 || bound > 0 && got >= bound || bound <= 0 && got != 0 {
				t.Fatalf("Jitter(%v) = %v, want in [0, %v)", bound, got, max(bound, 0))
			}
		}
	}
}
