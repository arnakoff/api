// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"testing/iotest"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// newKeyAdmin returns a KeyAdmin over store, at t0, drawing from
// crypto/rand.
func newKeyAdmin(store *keyStore) *app.KeyAdmin {
	return app.NewKeyAdmin(store, keySecret, &clock{now: t0}, rand.Reader)
}

func TestKeyAdminIssue(t *testing.T) {
	t.Parallel()

	store := &keyStore{}
	key, rec, err := newKeyAdmin(store).Issue(context.Background(), domain.KeyLive, "  Bot Discord  ")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	want := domain.KeyRecord{
		ID: 1, Digest: domain.DigestAPIKey(keySecret, key), Environment: domain.KeyLive,
		Label: "Bot Discord", Hint: key.Hint(), CreatedAt: t0,
	}
	if rec != want {
		t.Errorf("Issue() record = %+v, want %+v", rec, want)
	}
	if key.Environment() != domain.KeyLive {
		t.Errorf("Issue() key environment = %v, want live", key.Environment())
	}
}

func TestKeyAdminIssueErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &keyStore{}
	if _, _, err := newKeyAdmin(store).Issue(ctx, domain.KeyLive, " "); !errors.Is(err, domain.ErrInvalidKeyLabel) {
		t.Errorf("Issue() with a blank label error = %v, want ErrInvalidKeyLabel", err)
	}
	broken := app.NewKeyAdmin(store, keySecret, &clock{now: t0}, iotest.ErrReader(errors.New("no entropy")))
	if _, _, err := broken.Issue(ctx, domain.KeyLive, "bot"); err == nil {
		t.Error("Issue() with a failing random source: no error")
	}
	failing := &keyStore{err: errors.New("connection refused")}
	if _, _, err := newKeyAdmin(failing).Issue(ctx, domain.KeyLive, "bot"); !errors.Is(err, failing.err) {
		t.Errorf("Issue() with a failing store error = %v, want it wrapped", err)
	}
	if len(store.recs) != 0 {
		t.Errorf("store holds %d records after failed issues, want 0", len(store.recs))
	}
}

func TestKeyAdminListAndRevoke(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &keyStore{}
	clk := &clock{now: t0}
	admin := app.NewKeyAdmin(store, keySecret, clk, rand.Reader)
	for _, label := range []string{"a", "b"} {
		if _, _, err := admin.Issue(ctx, domain.KeyTest, label); err != nil {
			t.Fatal(err)
		}
	}
	clk.advance(time.Hour)
	if err := admin.Revoke(ctx, 2); err != nil {
		t.Fatalf("Revoke(2) error = %v", err)
	}
	if err := admin.Revoke(ctx, 9); !errors.Is(err, app.ErrKeyNotFound) {
		t.Errorf("Revoke(9) error = %v, want ErrKeyNotFound", err)
	}
	recs, err := admin.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(recs) != 2 || !recs[0].Active() || !recs[1].RevokedAt.Equal(t0.Add(time.Hour)) {
		t.Errorf("List() = %+v, want key 1 active and key 2 revoked at t0+1h", recs)
	}
}

func TestAuthenticate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &keyStore{}
	admin := newKeyAdmin(store)
	live, rec, err := admin.Issue(ctx, domain.KeyLive, "live")
	if err != nil {
		t.Fatal(err)
	}
	revoked, revokedRec, err := admin.Issue(ctx, domain.KeyLive, "revoked")
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Revoke(ctx, revokedRec.ID); err != nil {
		t.Fatal(err)
	}
	test, _, err := admin.Issue(ctx, domain.KeyTest, "test")
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := domain.NewAPIKey(domain.KeyLive, bytes.NewReader(bytes.Repeat([]byte{7}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	auth := app.NewAuthenticator(store, keySecret, domain.KeyLive)

	got, err := auth.Authenticate(ctx, live.String())
	if err != nil || got != rec {
		t.Errorf("Authenticate(live key) = %+v, %v, want %+v", got, err, rec)
	}
	tests := []struct {
		name string
		raw  string
		want error
	}{
		{"malformed", "ark_live_nope", domain.ErrInvalidAPIKey},
		{"other environment", test.String(), app.ErrKeyNotFound},
		{"unknown", unknown.String(), app.ErrKeyNotFound},
		{"revoked", revoked.String(), app.ErrKeyRevoked},
	}
	for _, tt := range tests {
		if _, err := auth.Authenticate(ctx, tt.raw); !errors.Is(err, tt.want) {
			t.Errorf("Authenticate(%s) error = %v, want %v", tt.name, err, tt.want)
		}
	}
}

func TestAuthenticateStoreFailure(t *testing.T) {
	t.Parallel()

	key, err := domain.NewAPIKey(domain.KeyLive, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	store := &keyStore{err: errors.New("connection refused")}
	_, err = app.NewAuthenticator(store, keySecret, domain.KeyLive).Authenticate(context.Background(), key.String())
	if !errors.Is(err, store.err) || errors.Is(err, app.ErrKeyNotFound) {
		t.Errorf("Authenticate() error = %v, want the store error, not ErrKeyNotFound", err)
	}
}

// wrongFinder answers every digest with a record of another digest, as a
// store matching on a faulty index would.
type wrongFinder struct{}

func (wrongFinder) FindKey(context.Context, domain.KeyDigest) (domain.KeyRecord, error) {
	return domain.KeyRecord{ID: 1, Digest: domain.KeyDigest{1}, Environment: domain.KeyLive}, nil
}

func TestAuthenticateRechecksDigest(t *testing.T) {
	t.Parallel()

	key, err := domain.NewAPIKey(domain.KeyLive, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := app.NewAuthenticator(wrongFinder{}, keySecret, domain.KeyLive)
	if _, err := auth.Authenticate(context.Background(), key.String()); !errors.Is(err, app.ErrKeyNotFound) {
		t.Errorf("Authenticate() error = %v, want ErrKeyNotFound", err)
	}
}
