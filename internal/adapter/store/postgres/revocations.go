// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arnakoff/api/internal/domain"
)

// WatchApplicationName is the application_name of the connection of
// WatchRevocations, as pg_stat_activity shows it.
const WatchApplicationName = "arnakoff revocation watch"

// revokedChannel is the channel on which migration 0003 announces each
// changed or deleted key, with its hex digest.
const revokedChannel = "api_key_revoked"

// closeTimeout bounds the closing of the connection of the watch.
const closeTimeout = 3 * time.Second

// RevocationSink is told of the keys revoked in the store: Forget for each
// key, Resume once every later revocation will be told, Suspend as soon as
// one may be missed. app.KeyCache is one.
type RevocationSink interface {
	Forget(d domain.KeyDigest)
	Suspend()
	Resume()
}

// WatchRevocations tells sink of the keys revoked in the store until ctx
// is done, on a connection of its own. When the connection fails, it
// suspends sink, reports the failure to report and connects again after
// retry. sink is suspended when it returns.
func (k *Keys) WatchRevocations(ctx context.Context, sink RevocationSink, retry time.Duration, report func(error)) {
	for {
		err := k.listen(ctx, sink)
		sink.Suspend()
		if ctx.Err() != nil {
			return
		}
		report(err)
		t := time.NewTimer(retry)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

// listen connects, listens to revokedChannel, resumes sink and tells it of
// each revocation until the connection fails or ctx is done.
func (k *Keys) listen(ctx context.Context, sink RevocationSink) error {
	cfg := k.pool.Config().ConnConfig
	cfg.RuntimeParams["application_name"] = WatchApplicationName
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("watch revocations: connect: %w", err)
	}
	defer func() {
		closing, cancel := context.WithTimeout(context.WithoutCancel(ctx), closeTimeout)
		defer cancel()
		_ = conn.Close(closing)
	}()
	if _, err := conn.Exec(ctx, "LISTEN "+revokedChannel); err != nil {
		return fmt.Errorf("watch revocations: listen: %w", err)
	}
	sink.Resume()
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("watch revocations: %w", err)
		}
		d, err := parseDigest(n.Payload)
		if err != nil {
			return fmt.Errorf("watch revocations: %w", err)
		}
		sink.Forget(d)
	}
}

// errPayload means a notification on revokedChannel is not a key digest:
// the watch starts over, which suspends the cache meanwhile.
var errPayload = errors.New("notification is not a key digest")

// parseDigest returns the KeyDigest of a hex payload.
func parseDigest(payload string) (domain.KeyDigest, error) {
	var d domain.KeyDigest
	b, err := hex.DecodeString(payload)
	if err != nil || len(b) != len(d) {
		return domain.KeyDigest{}, errPayload
	}
	copy(d[:], b)
	return d, nil
}
