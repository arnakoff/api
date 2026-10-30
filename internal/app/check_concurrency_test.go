// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// patience bounds how long a test waits for what should happen at once.
const patience = 5 * time.Second

func TestCheckConsultsTheRegistriesConcurrently(t *testing.T) {
	t.Parallel()

	// Each company registry waits until the other one was asked: consulted
	// one after the other, the first would wait until the deadline.
	var started sync.WaitGroup
	started.Add(2)
	meet := func(ctx context.Context) {
		started.Done()
		waitOrDone(ctx, &started)
	}
	r := newRegistries()
	r.companies.hold, r.notices.hold = meet, meet
	ctx, cancel := context.WithTimeout(context.Background(), patience)
	defer cancel()

	got := app.NewCheck(r.all(), &clock{now: t0}).Check(ctx, parse(t, "120027016")).Sources()

	if want := states(naState, okState, okState); !slices.Equal(got, want) {
		t.Errorf("Sources() = %v, want %v", got, want)
	}
}

func TestCheckReturnsPartialResultsAtTheDeadline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(app.Registries) *app.Check
		ctx   func() (context.Context, context.CancelFunc)
	}{
		{
			"request deadline",
			func(r app.Registries) *app.Check { return app.NewCheck(r, &clock{now: t0}) },
			func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 20*time.Millisecond)
			},
		},
		{
			"check timeout",
			func(r app.Registries) *app.Check {
				return app.NewCheck(r, &clock{now: t0}, app.WithTimeout(20*time.Millisecond))
			},
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// The notice registry ignores its context and answers too late.
			late := make(chan struct{})
			defer close(late)
			r := newRegistries()
			r.notices.hold = func(context.Context) { <-late }
			ctx, cancel := tt.ctx()
			defer cancel()

			done := make(chan domain.Report, 1)
			go func() { done <- tt.check(r.all()).Check(ctx, parse(t, "120027016")) }()

			select {
			case got := <-done:
				want := states(naState, okState, domain.SourceUnavailable)
				if !slices.Equal(got.Sources(), want) {
					t.Errorf("Sources() = %v, want %v", got.Sources(), want)
				}
			case <-time.After(patience):
				t.Fatal("Check() still waits for a registry past its deadline")
			}
		})
	}
}

func TestCheckTimeoutKeepsAnEarlierRequestDeadline(t *testing.T) {
	t.Parallel()

	var deadline time.Time
	r := newRegistries()
	r.domains.hold = func(ctx context.Context) { deadline, _ = ctx.Deadline() }
	want := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), want)
	defer cancel()

	app.NewCheck(r.all(), &clock{now: t0}, app.WithTimeout(time.Hour)).Check(ctx, parse(t, "exemple.fr"))

	if !deadline.Equal(want) {
		t.Errorf("registry deadline = %v, want the request's %v", deadline, want)
	}
}

// waitOrDone returns once wg is done or ctx is.
func waitOrDone(ctx context.Context, wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
