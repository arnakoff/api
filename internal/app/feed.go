// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// SourceSync is the sync state of one source, as /v1/sources shows it.
type SourceSync struct {
	Source domain.Source
	// LastSuccess is when the snapshot in service was fetched, zero when
	// none is.
	LastSuccess time.Time
	// LastFailure is when the last failed or rejected sync ran, zero when
	// none did. It is kept after a later success.
	LastFailure time.Time
	// LastError is why that sync failed.
	LastError error
	// Rows is the number of rows of the snapshot in service.
	Rows int
	// Newest is the date of the most recent entry in service, zero when the
	// source dates none.
	Newest time.Time
}

// feed is the sync state of one source. Readers load current and failure
// without locking; writers hold run and swap in new values.
type feed struct {
	fetcher SourceFetcher
	spec    domain.SnapshotSpec

	run     sync.Mutex // one sync or restore of the source at a time
	current atomic.Pointer[domain.Snapshot]
	failure atomic.Pointer[failure]
}

// failure is a sync that failed.
type failure struct {
	at  time.Time
	err error
}

// rows returns the number of rows in service, 0 when none is.
func (f *feed) rows() int {
	if snap := f.current.Load(); snap != nil {
		return len(snap.Rows)
	}
	return 0
}

// sourceSync returns the state of the feed.
func (f *feed) sourceSync() SourceSync {
	st := SourceSync{Source: f.spec.Source}
	if snap := f.current.Load(); snap != nil {
		st.LastSuccess = snap.FetchedAt
		st.Rows = len(snap.Rows)
		st.Newest = snap.Newest
	}
	if fail := f.failure.Load(); fail != nil {
		st.LastFailure = fail.at
		st.LastError = fail.err
	}
	return st
}
