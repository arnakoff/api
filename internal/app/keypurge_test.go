// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// revokedAt returns an issued record of digest {b}, revoked at at; active
// when at is zero.
func revokedAt(b byte, at time.Time) domain.KeyRecord {
	return domain.KeyRecord{
		ID: int64(b), Digest: domain.KeyDigest{b}, Environment: domain.KeyLive,
		Label: "Bot Discord", Hint: "a1B2", CreatedAt: t0.AddDate(-3, 0, 0), RevokedAt: at,
	}
}

func TestKeyPurgeOnceDeletesKeysRevokedLongerThanTheRetention(t *testing.T) {
	t.Parallel()

	ks := &keyStore{recs: []domain.KeyRecord{
		revokedAt(1, time.Time{}),
		revokedAt(2, t0.Add(-domain.RevokedKeyRetention-time.Second)),
		revokedAt(3, t0.Add(-domain.RevokedKeyRetention)),
		revokedAt(4, t0.Add(-time.Hour)),
	}}
	p := app.NewKeyPurge(ks, &clock{now: t0}, &sleeper{clock: &clock{now: t0}, stop: after(0)}, time.Hour)

	n, err := p.Once(t.Context())
	if err != nil || n != 1 {
		t.Fatalf("Once() = %d, %v; want 1 key deleted", n, err)
	}
	recs, _ := ks.ListKeys(t.Context())
	var ids []int64
	for _, r := range recs {
		ids = append(ids, r.ID)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 3 || ids[2] != 4 {
		t.Errorf("keys left = %v, want [1 3 4]", ids)
	}
}

func TestKeyPurgeOnceWrapsTheStoreError(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	p := app.NewKeyPurge(&keyStore{err: down}, &clock{now: t0}, &sleeper{clock: &clock{now: t0}, stop: after(0)}, time.Hour)

	if _, err := p.Once(t.Context()); !errors.Is(err, down) {
		t.Errorf("Once() error = %v, want it to wrap %v", err, down)
	}
}

func TestKeyPurgeRunPurgesAtOnceThenAtEachInterval(t *testing.T) {
	t.Parallel()

	clk := &clock{now: t0}
	ks := &keyStore{recs: []domain.KeyRecord{
		revokedAt(1, t0.Add(-domain.RevokedKeyRetention-time.Second)),
		revokedAt(2, t0.Add(-domain.RevokedKeyRetention+time.Hour/2)),
	}}
	sl := &sleeper{clock: clk, stop: after(1)}
	var got []int64
	report := func(_ context.Context, n int64, err error) {
		if err != nil {
			t.Errorf("report() error = %v", err)
		}
		got = append(got, n)
	}

	app.NewKeyPurge(ks, clk, sl, time.Hour).Run(t.Context(), report)

	if len(got) != 2 || got[0] != 1 || got[1] != 1 {
		t.Errorf("reported = %v, want [1 1]: one key at start, one an hour later", got)
	}
	if p := sl.pauses(); len(p) != 1 || p[0] != time.Hour {
		t.Errorf("pauses = %v, want one of an hour", p)
	}
}

func TestKeyPurgeRunReportsFailuresAndGoesOn(t *testing.T) {
	t.Parallel()

	clk := &clock{now: t0}
	ks := &keyStore{err: errors.New("database down")}
	var failures int
	report := func(_ context.Context, _ int64, err error) {
		if err != nil {
			failures++
		}
	}

	app.NewKeyPurge(ks, clk, &sleeper{clock: clk, stop: after(2)}, time.Hour).Run(t.Context(), report)

	if failures != 3 {
		t.Errorf("failures reported = %d, want 3", failures)
	}
}

func TestKeyPurgeRunStopsSilentlyWhenCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	clk := &clock{now: t0}
	ks := &keyStore{err: context.Canceled}
	report := func(_ context.Context, n int64, err error) {
		t.Errorf("report(%d, %v) after cancellation, want none", n, err)
	}

	app.NewKeyPurge(ks, clk, &sleeper{clock: clk, stop: after(5)}, time.Hour).Run(ctx, report)
}
