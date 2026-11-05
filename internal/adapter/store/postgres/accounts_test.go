// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// The store erases accounts.
var _ app.AccountEraser = (*postgres.Keys)(nil)

func TestEraseAccountDeletesTheKeysAndLoginRequestsOfTheAccountOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keys := openKeys(t)
	erased, kept := seedAccounts(t, keys)
	s := watch(t, keys)

	got, err := keys.EraseAccount(ctx, mustEmail(t, "dev@EXEMPLE.fr"))
	if err != nil {
		t.Fatalf("EraseAccount() error = %v", err)
	}
	if want := (app.PurgeCount{Keys: 2, LoginRequests: 2}); got != want {
		t.Errorf("EraseAccount() = %+v, want %+v", got, want)
	}
	awaitForget(t, s, erased[1])
	for _, d := range erased {
		if _, err := keys.FindKey(ctx, d); !errors.Is(err, app.ErrKeyNotFound) {
			t.Errorf("FindKey(erased) error = %v, want ErrKeyNotFound", err)
		}
	}
	for _, d := range kept {
		if _, err := keys.FindKey(ctx, d); err != nil {
			t.Errorf("FindKey(kept) error = %v", err)
		}
	}
	if _, err := keys.UseLoginRequest(ctx, domain.TokenDigest{0xE0, 0}, created); !errors.Is(err, app.ErrLoginLinkInvalid) {
		t.Errorf("UseLoginRequest(erased) error = %v, want ErrLoginLinkInvalid", err)
	}
	if _, err := keys.UseLoginRequest(ctx, domain.TokenDigest{0xE0, 2}, created); err != nil {
		t.Errorf("UseLoginRequest(other account) error = %v", err)
	}

	again, err := keys.EraseAccount(ctx, mustEmail(t, "dev@exemple.fr"))
	if err != nil || again != (app.PurgeCount{}) {
		t.Errorf("EraseAccount() again = %+v, %v; want nothing deleted and no error", again, err)
	}
}

// seedAccounts stores two keys of dev@exemple.fr, revoked then active,
// in two cases, a key of another account and an operator's key, and a
// login request to each address, of digest {0xE0, i}: those of
// dev@exemple.fr first. It returns the digests of the keys of
// dev@exemple.fr, the active one last, and of the other keys.
func seedAccounts(t *testing.T, keys *postgres.Keys) (erased, kept []domain.KeyDigest) {
	t.Helper()
	ctx := context.Background()
	replace := func(r domain.KeyRecord) domain.KeyDigest {
		rec, err := keys.ReplaceAccountKey(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		return rec.Digest
	}
	erased = []domain.KeyDigest{
		replace(accountRecord(0xE1, mustEmail(t, "Dev@Exemple.fr"), created)),
		replace(accountRecord(0xE2, mustEmail(t, "dev@exemple.fr"), created.Add(time.Hour))),
	}
	operator, err := keys.CreateKey(ctx, record(0xE0, domain.KeyLive))
	if err != nil {
		t.Fatal(err)
	}
	kept = []domain.KeyDigest{operator.Digest, replace(accountRecord(0xE3, mustEmail(t, "autre@exemple.fr"), created))}
	for i, to := range []string{"DEV@exemple.fr", "dev@exemple.fr", "autre@exemple.fr"} {
		r := domain.NewLoginRequest(domain.TokenDigest{0xE0, byte(i)}, mustEmail(t, to), created)
		if err := keys.CreateLoginRequest(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	return erased, kept
}
