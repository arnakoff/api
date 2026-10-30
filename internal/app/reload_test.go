// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
)

func TestReloaderSwapsInWhatTheSyncProcessStored(t *testing.T) {
	t.Parallel()

	st := newStore()
	s, clk := newSync(t, st, newFetcher(amfSpec))
	sl := &sleeper{clock: clk, stop: after(1), before: func(int) {
		st.mu.Lock()
		defer st.mu.Unlock()
		st.saved["AMF"] = stored(70, t0)
	}}

	app.NewReloader(s, sl, &alerter{}, time.Minute).Run(t.Context())

	if got, ok := s.Snapshot("AMF"); !ok || len(got.Rows) != 70 {
		t.Errorf("Snapshot() = %d rows, %t; want the 70 rows stored meanwhile", len(got.Rows), ok)
	}
	if got := sl.pauses(); len(got) != 1 || got[0] != time.Minute {
		t.Errorf("pauses = %v, want one of a minute", got)
	}
}

func TestReloaderAlertsOncePerFailingSpell(t *testing.T) {
	t.Parallel()

	st := newStore()
	s, clk := newSync(t, st, newFetcher(amfSpec))
	// Reloads 0 and 1 fail, 2 succeeds, 3 fails again.
	failing := []bool{true, true, false, true}
	sl := &sleeper{clock: clk, stop: after(len(failing)), before: func(i int) {
		if failing[i] {
			st.setLoadErr(errDisk)
			return
		}
		st.setLoadErr(nil)
	}}
	al := &alerter{}

	app.NewReloader(s, sl, al, time.Minute).Run(t.Context())

	got := al.received()
	if len(got) != 2 {
		t.Fatalf("alerts = %+v, want one per failing spell, 2", got)
	}
	for _, a := range got {
		if a.Source != "AMF" || !errors.Is(a.Err, errDisk) {
			t.Errorf("alert = %+v, want AMF for %v", a, errDisk)
		}
	}
	if want := t0.Add(time.Minute); !got[0].At.Equal(want) {
		t.Errorf("first alert at %v, want %v", got[0].At, want)
	}
}
