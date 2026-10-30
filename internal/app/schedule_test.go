// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

const every = 6 * time.Hour

func noJitter(time.Duration) time.Duration { return 0 }

func newScheduler(t *testing.T, s *app.Sync, sl *sleeper, al *alerter, jitter app.Jitter, plans ...app.Plan) *app.Scheduler {
	t.Helper()
	sc, err := app.NewScheduler(s, sl, al, jitter, plans...)
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	return sc
}

func TestSchedulerSyncsAtOnceWithoutStoredSnapshotThenEveryInterval(t *testing.T) {
	t.Parallel()

	f := newFetcher(amfSpec).then(snapshot(10, time.Time{}), nil).then(snapshot(11, time.Time{}), nil)
	s, clk := newSync(t, newStore(), f)
	sl := &sleeper{clock: clk, stop: after(2)}
	var asked time.Duration
	sc := newScheduler(t, s, sl, &alerter{}, fixedJitter(time.Minute, &asked), app.Plan{Source: "AMF", Every: every})

	sc.Run(t.Context())
	if got, want := sl.pauses(), []time.Duration{0, every + time.Minute}; !slices.Equal(got, want) {
		t.Errorf("pauses = %v, want %v", got, want)
	}
	if asked != every/10 {
		t.Errorf("jitter asked up to %v, want a tenth of the interval, %v", asked, every/10)
	}
	if got := state(t, s, "AMF"); got.Rows != 11 || !got.LastSuccess.Equal(t0.Add(every+time.Minute)) {
		t.Errorf("state = %+v, want the second snapshot", got)
	}
}

func TestSchedulerWaitsUntilTheStoredSnapshotIsDue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fetchedAt time.Time
		want      time.Duration
	}{
		{"fresh", t0.Add(-time.Hour), every - time.Hour},
		{"just due", t0.Add(-every), 0},
		{"overdue", t0.Add(-3 * every), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			st := newStore()
			st.saved["AMF"] = stored(10, tt.fetchedAt)
			s, clk := newSync(t, st, newFetcher(amfSpec))
			if err := s.Restore(t.Context()); err != nil {
				t.Fatalf("Restore: %v", err)
			}
			sl := &sleeper{clock: clk, stop: after(0)}
			sc := newScheduler(t, s, sl, &alerter{}, noJitter, app.Plan{Source: "AMF", Every: every})

			sc.Run(t.Context())
			if got := sl.firstAsked(); got != tt.want {
				t.Errorf("first pause = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchedulerAlertsOnFailureAndKeepsGoing(t *testing.T) {
	t.Parallel()

	f := newFetcher(amfSpec).then(domain.Snapshot{}, errUpstream).then(snapshot(10, time.Time{}), nil)
	s, clk := newSync(t, newStore(), f)
	sl := &sleeper{clock: clk, stop: after(2)}
	al := &alerter{}
	sc := newScheduler(t, s, sl, al, noJitter, app.Plan{Source: "AMF", Every: every})

	sc.Run(t.Context())
	got := al.received()
	if len(got) != 1 || got[0].Source != "AMF" || !got[0].At.Equal(t0) || !errors.Is(got[0].Err, errUpstream) {
		t.Fatalf("alerts = %+v, want one AMF alert at %v for %v", got, t0, errUpstream)
	}
	if _, ok := s.Snapshot("AMF"); !ok {
		t.Error("the next run after the failure did not sync")
	}
}

func TestSchedulerRunsEverySource(t *testing.T) {
	t.Parallel()

	s, clk := newSync(t, newStore(),
		newFetcher(amfSpec).then(snapshot(10, time.Time{}), nil),
		newFetcher(acprSpec).then(acprSnapshot(5), nil),
	)
	sl := &sleeper{clock: clk, stop: immediateOnly}
	sc := newScheduler(t, s, sl, &alerter{}, noJitter,
		app.Plan{Source: "AMF", Every: every}, app.Plan{Source: "ACPR", Every: 24 * time.Hour})

	sc.Run(t.Context())
	for _, src := range []domain.Source{"AMF", "ACPR"} {
		if _, ok := s.Snapshot(src); !ok {
			t.Errorf("%s was not synced", src)
		}
	}
}

func TestSchedulerStopsWithTheContext(t *testing.T) {
	t.Parallel()

	s, clk := newSync(t, newStore(), newFetcher(amfSpec))
	sc := newScheduler(t, s, &sleeper{clock: clk, stop: after(10)}, &alerter{}, noJitter, app.Plan{Source: "AMF", Every: every})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	sc.Run(ctx)
	if _, ok := s.Snapshot("AMF"); ok {
		t.Error("Run() synced after the context was done")
	}
}

func TestNewSchedulerRejectsBadPlans(t *testing.T) {
	t.Parallel()

	s, clk := newSync(t, newStore(), newFetcher(amfSpec))
	tests := []struct {
		name  string
		plans []app.Plan
		want  error
	}{
		{"unknown source", []app.Plan{{Source: "BODACC", Every: every}}, app.ErrUnknownSource},
		{"twice", []app.Plan{{Source: "AMF", Every: every}, {Source: "AMF", Every: every}}, app.ErrDuplicateSource},
		{"no interval", []app.Plan{{Source: "AMF"}}, app.ErrInterval},
		{"negative interval", []app.Plan{{Source: "AMF", Every: -every}}, app.ErrInterval},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := app.NewScheduler(s, &sleeper{clock: clk, stop: after(0)}, &alerter{}, noJitter, tt.plans...)
			if !errors.Is(err, tt.want) {
				t.Errorf("NewScheduler() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSchedulerOnceSyncsTheSourcesAskedAndAlertsOnFailure(t *testing.T) {
	t.Parallel()

	s, clk := newSync(t, newStore(),
		newFetcher(amfSpec).then(domain.Snapshot{}, errUpstream).then(snapshot(10, time.Time{}), nil),
		newFetcher(acprSpec),
	)
	al := &alerter{}
	sc := newScheduler(t, s, &sleeper{clock: clk, stop: after(0)}, al, noJitter,
		app.Plan{Source: "AMF", Every: every}, app.Plan{Source: "ACPR", Every: every})

	if err := sc.Once(t.Context(), "AMF"); !errors.Is(err, errUpstream) {
		t.Errorf("Once(AMF) = %v, want %v", err, errUpstream)
	}
	if got := al.received(); len(got) != 1 || got[0].Source != "AMF" {
		t.Errorf("alerts = %+v, want one for AMF", got)
	}
	if err := sc.Once(t.Context(), "AMF"); err != nil {
		t.Errorf("second Once(AMF) = %v", err)
	}
	if _, ok := s.Snapshot("ACPR"); ok {
		t.Error("Once(AMF) synced ACPR")
	}
}

func TestSchedulerOnceWithoutSourceSyncsThemAll(t *testing.T) {
	t.Parallel()

	s, clk := newSync(t, newStore(),
		newFetcher(amfSpec).then(snapshot(10, time.Time{}), nil),
		newFetcher(acprSpec).then(acprSnapshot(5), nil),
	)
	sc := newScheduler(t, s, &sleeper{clock: clk, stop: after(0)}, &alerter{}, noJitter,
		app.Plan{Source: "AMF", Every: every}, app.Plan{Source: "ACPR", Every: every})

	if err := sc.Once(t.Context()); err != nil {
		t.Fatalf("Once() = %v", err)
	}
	for _, src := range []domain.Source{"AMF", "ACPR"} {
		if _, ok := s.Snapshot(src); !ok {
			t.Errorf("%s was not synced", src)
		}
	}
}

func TestSchedulerOnceRejectsAnUnknownSourceBeforeSyncing(t *testing.T) {
	t.Parallel()

	s, clk := newSync(t, newStore(), newFetcher(amfSpec))
	al := &alerter{}
	sc := newScheduler(t, s, &sleeper{clock: clk, stop: after(0)}, al, noJitter, app.Plan{Source: "AMF", Every: every})

	if err := sc.Once(t.Context(), "AMF", "BODACC"); !errors.Is(err, app.ErrUnknownSource) {
		t.Errorf("Once(AMF, BODACC) = %v, want %v", err, app.ErrUnknownSource)
	}
	if len(al.received()) != 0 {
		t.Error("an unknown source raised an alert")
	}
}
