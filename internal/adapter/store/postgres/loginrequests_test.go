// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// The store implements the ports of the key requests.
var (
	_ app.LoginRequestStore = (*postgres.Keys)(nil)
	_ app.AccountKeyIssuer  = (*postgres.Keys)(nil)
)

func mustEmail(t *testing.T, s string) domain.Email {
	t.Helper()
	e, err := domain.ParseEmail(s)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestLoginRequestsAreUsedOnceWithinTheirLifetime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	to := mustEmail(t, "Dev@Exemple.fr")
	r := domain.NewLoginRequest(domain.TokenDigest{1}, to, created)
	if err := keys.CreateLoginRequest(ctx, r); err != nil {
		t.Fatalf("CreateLoginRequest() error = %v", err)
	}
	if err := keys.CreateLoginRequest(ctx, r); err == nil {
		t.Error("CreateLoginRequest() of a digest already stored: no error")
	}

	if _, err := keys.UseLoginRequest(ctx, domain.TokenDigest{2}, created); !errors.Is(err, app.ErrLoginLinkInvalid) {
		t.Errorf("UseLoginRequest(unknown) error = %v, want ErrLoginLinkInvalid", err)
	}
	if _, err := keys.UseLoginRequest(ctx, r.Digest, r.ExpiresAt); !errors.Is(err, app.ErrLoginLinkInvalid) {
		t.Errorf("UseLoginRequest(at expiry) error = %v, want ErrLoginLinkInvalid", err)
	}
	got, err := keys.UseLoginRequest(ctx, r.Digest, r.ExpiresAt.Add(-time.Microsecond))
	if err != nil || got != to {
		t.Fatalf("UseLoginRequest() = %v, %v; want the address", got.Redacted(), err)
	}
	if _, err := keys.UseLoginRequest(ctx, r.Digest, r.ExpiresAt.Add(-time.Microsecond)); !errors.Is(err, app.ErrLoginLinkInvalid) {
		t.Errorf("UseLoginRequest(used) error = %v, want ErrLoginLinkInvalid", err)
	}
}

func TestUseLoginRequestSucceedsOnceUnderConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	r := domain.NewLoginRequest(domain.TokenDigest{1}, mustEmail(t, "dev@exemple.fr"), created)
	if err := keys.CreateLoginRequest(ctx, r); err != nil {
		t.Fatal(err)
	}
	var (
		wg sync.WaitGroup
		mu sync.Mutex
		ok int
	)
	for range 8 {
		wg.Go(func() {
			if _, err := keys.UseLoginRequest(ctx, r.Digest, created.Add(time.Minute)); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if ok != 1 {
		t.Errorf("%d uses succeeded, want 1", ok)
	}
}

func TestPurgeLoginRequestsDeletesThoseEndedBeforeTheCutoff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	to := mustEmail(t, "dev@exemple.fr")
	cutoff := created.Add(48 * time.Hour)
	expired := domain.NewLoginRequest(domain.TokenDigest{1}, to, cutoff.Add(-domain.LoginLinkLifetime-time.Microsecond))
	used := domain.NewLoginRequest(domain.TokenDigest{2}, to, cutoff.Add(-time.Minute))
	open := domain.NewLoginRequest(domain.TokenDigest{3}, to, cutoff.Add(-time.Minute))
	for _, r := range []domain.LoginRequest{expired, used, open} {
		if err := keys.CreateLoginRequest(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := keys.UseLoginRequest(ctx, used.Digest, cutoff.Add(-time.Microsecond)); err != nil {
		t.Fatal(err)
	}

	n, err := keys.PurgeLoginRequests(ctx, cutoff)
	if err != nil || n != 2 {
		t.Fatalf("PurgeLoginRequests() = %d, %v; want 2 deleted", n, err)
	}
	if _, err := keys.UseLoginRequest(ctx, open.Digest, cutoff); err != nil {
		t.Errorf("UseLoginRequest(kept) error = %v", err)
	}
}

// accountRecord returns a self-service record of digest {b} for to, issued
// at at.
func accountRecord(b byte, to domain.Email, at time.Time) domain.KeyRecord {
	return domain.KeyRecord{
		Digest: domain.KeyDigest{b}, Environment: domain.KeyLive, Label: app.SelfServiceKeyLabel,
		Hint: "a1B2", Email: to, CreatedAt: at,
	}
}

func TestReplaceAccountKeyRevokesThePreviousKeyOfTheAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	operator, err := keys.CreateKey(ctx, record(9, domain.KeyLive))
	if err != nil {
		t.Fatal(err)
	}
	first, err := keys.ReplaceAccountKey(ctx, accountRecord(1, mustEmail(t, "dev@exemple.fr"), created))
	if err != nil {
		t.Fatalf("ReplaceAccountKey() error = %v", err)
	}
	other, err := keys.ReplaceAccountKey(ctx, accountRecord(2, mustEmail(t, "autre@exemple.fr"), created))
	if err != nil {
		t.Fatal(err)
	}
	later := created.Add(time.Hour)
	second, err := keys.ReplaceAccountKey(ctx, accountRecord(3, mustEmail(t, "DEV@exemple.fr"), later))
	if err != nil {
		t.Fatalf("ReplaceAccountKey() again error = %v", err)
	}

	recs, err := keys.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	state := map[int64]domain.KeyRecord{}
	for _, r := range recs {
		state[r.ID] = r
	}
	if !state[first.ID].RevokedAt.Equal(later) {
		t.Errorf("first key revoked at %v, want %v", state[first.ID].RevokedAt, later)
	}
	if !state[operator.ID].Active() || !state[other.ID].Active() || !state[second.ID].Active() {
		t.Errorf("keys = %+v, want the operator's, the other account's and the new one active", recs)
	}
	if got := state[second.ID].Email; got != mustEmail(t, "DEV@exemple.fr") {
		t.Errorf("new key's address = %v, want it kept as written", got.Redacted())
	}
}

func TestReplaceAccountKeyKeepsOneActiveKeyUnderConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	to := mustEmail(t, "dev@exemple.fr")
	var wg sync.WaitGroup
	for b := byte(1); b <= 6; b++ {
		wg.Go(func() {
			if _, err := keys.ReplaceAccountKey(ctx, accountRecord(b, to, created.Add(time.Duration(b)*time.Second))); err != nil {
				t.Errorf("ReplaceAccountKey(%d) error = %v", b, err)
			}
		})
	}
	wg.Wait()
	recs, err := keys.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, r := range recs {
		if r.Active() {
			active++
		}
	}
	if len(recs) != 6 || active != 1 {
		t.Errorf("%d keys, %d active; want 6 keys, 1 active", len(recs), active)
	}
}
