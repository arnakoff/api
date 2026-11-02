// SPDX-License-Identifier: AGPL-3.0-only

package replay_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/registry/replay"
)

// recordedTape returns a tape holding the answers of liveRegistries to
// exemple.fr and 120027016.
func recordedTape(t *testing.T) *replay.Tape {
	t.Helper()
	tape, err := replay.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load of an empty directory: %v", err)
	}
	mustLookup(t, tape.Registries(liveRegistries()), "exemple.fr", "120027016")
	return tape
}

func TestReplayedAnswersArriveAfterTheLatency(t *testing.T) {
	t.Parallel()

	const latency = 20 * time.Millisecond
	regs := recordedTape(t).Replayed(latency)
	ctx := context.Background()
	lookups := map[string]func() error{
		"domains": func() error {
			_, err := regs.Domains.Lookup(ctx, mustDomain(t, "exemple.fr"))
			return err
		},
		"companies": func() error {
			_, err := regs.Companies.Lookup(ctx, mustSIREN(t, "120027016"))
			return err
		},
		"notices": func() error {
			_, err := regs.Notices.Notices(ctx, mustSIREN(t, "120027016"))
			return err
		},
	}
	for name, lookup := range lookups {
		start := time.Now()
		err := lookup()
		if elapsed := time.Since(start); err != nil || elapsed < latency {
			t.Errorf("%s: err %v after %v, want the recorded answer after %v", name, err, elapsed, latency)
		}
	}
}

func TestReplayedLookupStopsWithItsContext(t *testing.T) {
	t.Parallel()

	regs := recordedTape(t).Replayed(time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := regs.Domains.Lookup(ctx, mustDomain(t, "exemple.fr"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Lookup with a cancelled context: err %v, want context.Canceled", err)
	}
}

func TestReplayedWithoutLatencyAnswersTheTape(t *testing.T) {
	t.Parallel()

	regs := recordedTape(t).Replayed(0)

	_, err := regs.Companies.Lookup(context.Background(), mustSIREN(t, "552100554"))

	if !errors.Is(err, replay.ErrNotRecorded) {
		t.Errorf("Lookup of an unrecorded SIREN: err %v, want ErrNotRecorded", err)
	}
}
