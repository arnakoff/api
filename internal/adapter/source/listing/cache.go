// SPDX-License-Identifier: AGPL-3.0-only

package listing

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// cache holds the index of the snapshot in service of one source. It builds
// it once per snapshot, at the first read after the snapshot is swapped in,
// and is safe for concurrent use.
type cache[T any] struct {
	snapshots Snapshots
	source    domain.Source
	build     func(domain.Snapshot) (T, error)

	mu      sync.Mutex // one indexing at a time
	current atomic.Pointer[index[T]]
}

// index is the index of the snapshot fetched at fetchedAt, or why it could
// not be built.
type index[T any] struct {
	fetchedAt time.Time
	list      T
	err       error
}

// get returns the index in service and when its snapshot was fetched,
// app.ErrNoSnapshot when none is, or the error of its build.
func (c *cache[T]) get() (T, time.Time, error) {
	snap, ok := c.snapshots.Snapshot(c.source)
	if !ok {
		var zero T
		return zero, time.Time{}, fmt.Errorf("list %s: %w", c.source, app.ErrNoSnapshot)
	}
	idx := c.current.Load()
	if idx == nil || !idx.fetchedAt.Equal(snap.FetchedAt) {
		idx = c.index(snap)
	}
	return idx.list, idx.fetchedAt, idx.err
}

// index builds and keeps the index of snap, unless another read did. While
// one read builds it, the others keep the previous index, if any, rather
// than wait.
func (c *cache[T]) index(snap domain.Snapshot) *index[T] {
	if !c.mu.TryLock() {
		if idx := c.current.Load(); idx != nil {
			return idx
		}
		c.mu.Lock()
	}
	defer c.mu.Unlock()
	if idx := c.current.Load(); idx != nil && idx.fetchedAt.Equal(snap.FetchedAt) {
		return idx
	}
	idx := &index[T]{fetchedAt: snap.FetchedAt}
	idx.list, idx.err = c.build(snap)
	if idx.err != nil {
		idx.err = fmt.Errorf("list %s: %w", c.source, idx.err)
	}
	c.current.Store(idx)
	return idx
}
