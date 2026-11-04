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
	p := app.NewKeyPurge(purgeStore{ks, &loginStore{}}, &clock{now: t0}, &sleeper{clock: &clock{now: t0}, stop: after(0)}, time.Hour)

	n, err := p.Once(t.Context())
	if err != nil || n != (app.PurgeCount{Keys: 1}) {
		t.Fatalf("Once() = %+v, %v; want 1 key deleted", n, err)
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
	p := app.NewKeyPurge(purgeStore{&keyStore{err: down}, &loginStore{err: down}}, &clock{now: t0}, &sleeper{clock: &clock{now: t0}, stop: after(0)}, time.Hour)

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
	report := func(_ context.Context, n app.PurgeCount, err error) {
		if err != nil {
			t.Errorf("report() error = %v", err)
		}
		got = append(got, n.Keys)
	}

	app.NewKeyPurge(purgeStore{ks, &loginStore{}}, clk, sl, time.Hour).Run(t.Context(), report)

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
	report := func(_ context.Context, _ app.PurgeCount, err error) {
		if err != nil {
			failures++
		}
	}

	app.NewKeyPurge(purgeStore{ks, &loginStore{}}, clk, &sleeper{clock: clk, stop: after(2)}, time.Hour).Run(t.Context(), report)

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
	report := func(_ context.Context, n app.PurgeCount, err error) {
		t.Errorf("report(%+v, %v) after cancellation, want none", n, err)
	}

	app.NewKeyPurge(purgeStore{ks, &loginStore{err: context.Canceled}}, clk, &sleeper{clock: clk, stop: after(5)}, time.Hour).Run(ctx, report)
}

func TestKeyPurgeOnceDeletesLoginRequestsPastTheirRetention(t *testing.T) {
	t.Parallel()

	to := mustEmail(t, "dev@exemple.fr")
	old := domain.NewLoginRequest(domain.TokenDigest{1}, to, t0.Add(-domain.LoginRequestRetention-domain.LoginLinkLifetime-time.Second))
	usedLongAgo := domain.NewLoginRequest(domain.TokenDigest{2}, to, t0.Add(-48*time.Hour))
	recent := domain.NewLoginRequest(domain.TokenDigest{3}, to, t0.Add(-domain.LoginRequestRetention))
	logins := &loginStore{reqs: []loginRow{
		{LoginRequest: old},
		{LoginRequest: usedLongAgo, usedAt: usedLongAgo.CreatedAt.Add(time.Minute)},
		{LoginRequest: recent},
	}}
	p := app.NewKeyPurge(purgeStore{&keyStore{}, logins}, &clock{now: t0}, &sleeper{clock: &clock{now: t0}, stop: after(0)}, time.Hour)

	n, err := p.Once(t.Context())
	if err != nil || n != (app.PurgeCount{LoginRequests: 2}) {
		t.Fatalf("Once() = %+v, %v; want 2 login requests deleted", n, err)
	}
	if left := logins.requests(); len(left) != 1 || left[0].Digest != recent.Digest {
		t.Errorf("left = %+v, want the recent request only", left)
	}
}

func TestKeyPurgeOncePurgesLoginRequestsWhenTheKeysFail(t *testing.T) {
	t.Parallel()

	to := mustEmail(t, "dev@exemple.fr")
	logins := &loginStore{reqs: []loginRow{{LoginRequest: domain.NewLoginRequest(domain.TokenDigest{1}, to, t0.AddDate(0, 0, -3))}}}
	down := errors.New("database down")
	p := app.NewKeyPurge(purgeStore{&keyStore{err: down}, logins}, &clock{now: t0}, &sleeper{clock: &clock{now: t0}, stop: after(0)}, time.Hour)

	n, err := p.Once(t.Context())
	if !errors.Is(err, down) || n.LoginRequests != 1 {
		t.Fatalf("Once() = %+v, %v; want 1 login request deleted and the key error", n, err)
	}
}
