// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/clock"
)

// The cache of the API is told of revocations.
var _ postgres.RevocationSink = (*app.KeyCache)(nil)

// sink is a postgres.RevocationSink reporting each call on its channels.
type sink struct {
	forgot  chan domain.KeyDigest
	resumed chan struct{}
	paused  chan struct{}
}

func newSink() *sink {
	return &sink{forgot: make(chan domain.KeyDigest, 64), resumed: make(chan struct{}, 8), paused: make(chan struct{}, 8)}
}

func (s *sink) Forget(d domain.KeyDigest) { s.forgot <- d }
func (s *sink) Resume()                   { s.resumed <- struct{}{} }
func (s *sink) Suspend()                  { s.paused <- struct{}{} }

// await fails t unless ch delivers within a few seconds.
func await[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("no %s within 5 s", what)
		var zero T
		return zero
	}
}

// awaitForget fails t unless s forgets d within a few seconds. The channel
// is shared by the database: other tests' keys are skipped.
func awaitForget(t *testing.T, s *sink, d domain.KeyDigest) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case got := <-s.forgot:
			if got == d {
				return
			}
		case <-deadline:
			t.Fatal("revoked key not forgotten within 5 s")
		}
	}
}

// watch runs keys.WatchRevocations into a new sink until the end of t, and
// waits until it listens.
func watch(t *testing.T, keys *postgres.Keys) *sink {
	t.Helper()
	s := newSink()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		keys.WatchRevocations(ctx, s, 10*time.Millisecond, func(error) {})
	}()
	t.Cleanup(func() { cancel(); <-done })
	await(t, s.resumed, "Resume")
	return s
}

func TestWatchRevocationsForgetsARevokedKey(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	keys := openKeys(t)
	admin := app.NewKeyAdmin(keys, []byte("a test secret of thirty-two byte"), clock.System{}, rand.Reader)
	_, rec, err := admin.Issue(ctx, domain.KeyTest, "bot")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	s := watch(t, keys)
	if err := admin.Revoke(ctx, rec.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	awaitForget(t, s, rec.Digest)
}

func TestWatchRevocationsForgetsTheKeyAReplacementRevokes(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	keys := openKeys(t)
	to, err := domain.ParseEmail("Someone@Example.org")
	if err != nil {
		t.Fatal(err)
	}
	old := domain.KeyRecord{Digest: domain.KeyDigest{7}, Environment: domain.KeyTest, Label: "self-service", Hint: "aaaa", Email: to, CreatedAt: created}
	if _, err := keys.ReplaceAccountKey(ctx, old); err != nil {
		t.Fatalf("ReplaceAccountKey() error = %v", err)
	}
	s := watch(t, keys)
	next := old
	next.Digest, next.CreatedAt = domain.KeyDigest{8}, created.Add(time.Minute)
	if _, err := keys.ReplaceAccountKey(ctx, next); err != nil {
		t.Fatalf("ReplaceAccountKey() again error = %v", err)
	}
	awaitForget(t, s, old.Digest)
}

// When its connection is lost, the watch suspends the cache, connects
// again and resumes it. Not parallel: it ends the watches of the other
// tests too.
func TestWatchRevocationsSuspendsUntilItListensAgain(t *testing.T) {
	keys := openKeys(t)
	s := watch(t, keys)
	terminateListeners(t)
	await(t, s.paused, "Suspend")
	await(t, s.resumed, "Resume after the reconnection")
}

func TestWatchRevocationsSuspendsWhenStopped(t *testing.T) {
	t.Parallel()

	keys := openKeys(t)
	s := newSink()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		keys.WatchRevocations(ctx, s, 10*time.Millisecond, func(error) {})
	}()
	await(t, s.resumed, "Resume")
	cancel()
	<-done
	await(t, s.paused, "Suspend")
}

// A notification that is not a key digest may hide a revocation: the watch
// suspends the cache and starts over. Not parallel: every watch of the
// database hears it.
func TestWatchRevocationsStartsOverOnAMalformedNotification(t *testing.T) {
	keys := openKeys(t)
	s := watch(t, keys)
	notify(t, "not a digest")
	await(t, s.paused, "Suspend")
	await(t, s.resumed, "Resume after starting over")
}
