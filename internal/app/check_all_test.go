// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestCheckAllReportsEachInputInOrder(t *testing.T) {
	t.Parallel()

	ins := []domain.Input{parse(t, "www.exemple.fr"), parse(t, "120027016"), parse(t, "+33612345678")}

	got := app.NewCheck(newRegistries().all(), &clock{now: t0}).CheckAll(t.Context(), ins)

	if len(got) != len(ins) {
		t.Fatalf("CheckAll() = %d reports, want %d", len(got), len(ins))
	}
	for i, in := range ins {
		if got[i].Input() != in.String() {
			t.Errorf("report %d is of %q, want %q", i, got[i].Input(), in.String())
		}
	}
}

func TestCheckAllOfNoInputIsEmpty(t *testing.T) {
	t.Parallel()

	if got := app.NewCheck(newRegistries().all(), &clock{now: t0}).CheckAll(t.Context(), nil); len(got) != 0 {
		t.Errorf("CheckAll(nil) = %v, want no report", got)
	}
}

func TestCheckAllChecksTheInputsConcurrently(t *testing.T) {
	t.Parallel()

	// Each check waits until the other one started: checked one after the
	// other, the first would wait until the deadline.
	var started sync.WaitGroup
	started.Add(2)
	r := newRegistries()
	r.domains.hold = func(ctx context.Context) {
		started.Done()
		waitOrDone(ctx, &started)
	}
	ctx, cancel := context.WithTimeout(context.Background(), patience)
	defer cancel()

	got := app.NewCheck(r.all(), &clock{now: t0}).CheckAll(ctx, []domain.Input{parse(t, "a.fr"), parse(t, "b.fr")})

	for i, rep := range got {
		if s := rep.Sources()[0].Status; s != okState {
			t.Errorf("report %d: RDAP %v, want %v", i, s, okState)
		}
	}
}

func TestCheckAllChecksAtMostTheLimitAtOnce(t *testing.T) {
	t.Parallel()

	const limit = 2
	var (
		mu            sync.Mutex
		running, most int
	)
	r := newRegistries()
	r.domains.hold = func(context.Context) {
		mu.Lock()
		running++
		most = max(most, running)
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
		mu.Lock()
		running--
		mu.Unlock()
	}
	ins := make([]domain.Input, 3*limit)
	for i := range ins {
		ins[i] = parse(t, "exemple.fr")
	}

	app.NewCheck(r.all(), &clock{now: t0}, app.WithBatchLimit(limit)).CheckAll(t.Context(), ins)

	mu.Lock()
	defer mu.Unlock()
	if most > limit {
		t.Errorf("%d checks at once, want at most %d", most, limit)
	}
}

func TestCheckAllWithALimitBelowOneChecksOneAtATime(t *testing.T) {
	t.Parallel()

	ins := []domain.Input{parse(t, "a.fr"), parse(t, "b.fr")}
	done := make(chan []domain.Report, 1)
	go func() {
		done <- app.NewCheck(newRegistries().all(), &clock{now: t0}, app.WithBatchLimit(0)).CheckAll(t.Context(), ins)
	}()

	select {
	case got := <-done:
		if len(got) != len(ins) {
			t.Errorf("CheckAll() = %d reports, want %d", len(got), len(ins))
		}
	case <-time.After(patience):
		t.Fatal("CheckAll() with a limit of 0 never returns")
	}
}

func TestCheckAllBoundsTheWholeBatchByTheTimeout(t *testing.T) {
	t.Parallel()

	// One at a time, each check waiting for its own timeout, the batch
	// would last four timeouts: past the server's write timeout for a full
	// batch of slow registries.
	const timeout = 100 * time.Millisecond
	r := newRegistries()
	r.domains.hold = func(ctx context.Context) { <-ctx.Done() }
	ins := []domain.Input{parse(t, "a.fr"), parse(t, "b.fr"), parse(t, "c.fr"), parse(t, "d.fr")}
	check := app.NewCheck(r.all(), &clock{now: t0}, app.WithTimeout(timeout), app.WithBatchLimit(1))

	start := time.Now()
	got := check.CheckAll(context.Background(), ins)

	if elapsed := time.Since(start); elapsed >= 3*timeout {
		t.Errorf("CheckAll() took %v, want about one timeout of %v", elapsed, timeout)
	}
	for i, rep := range got {
		if s := rep.Sources()[0].Status; s != domain.SourceUnavailable {
			t.Errorf("report %d: RDAP %v, want unavailable past the deadline", i, s)
		}
	}
}

func TestCheckAllConsultsNoRegistryPastTheDeadline(t *testing.T) {
	t.Parallel()

	// With one check at a time, the first holds the batch until the
	// deadline: the three inputs whose turn comes after it must not reach
	// the registries, or a timed-out batch would fire them all at once.
	const timeout = 50 * time.Millisecond
	r := newRegistries()
	r.domains.hold = func(ctx context.Context) { <-ctx.Done() }
	ins := []domain.Input{parse(t, "a.fr"), parse(t, "b.fr"), parse(t, "c.fr"), parse(t, "d.fr")}
	check := app.NewCheck(r.all(), &clock{now: t0}, app.WithTimeout(timeout), app.WithBatchLimit(1))

	got := check.CheckAll(context.Background(), ins)

	if asked := r.domains.got(); len(asked) != 1 {
		t.Errorf("RDAP consulted about %q, want only the input checked before the deadline", asked)
	}
	for i, rep := range got {
		if s := rep.Sources()[0].Status; s != domain.SourceUnavailable {
			t.Errorf("report %d: RDAP %v, want unavailable past the deadline", i, s)
		}
	}
}
