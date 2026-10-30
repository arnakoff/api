// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"

	"github.com/arnakoff/api/internal/domain"
)

// DryRun returns a store that reads st and keeps nothing: a Sync on it,
// once restored, validates new snapshots against the stored ones without
// replacing them.
func DryRun(st SnapshotStore) SnapshotStore {
	return dryRun{st}
}

type dryRun struct {
	SnapshotStore
}

// Save keeps nothing.
func (dryRun) Save(context.Context, domain.Snapshot) error {
	return nil
}
