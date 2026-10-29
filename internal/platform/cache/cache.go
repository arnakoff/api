// SPDX-License-Identifier: AGPL-3.0-only

package cache

import (
	"container/list"
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache maps string keys to values of type V. It is safe for concurrent use.
// A cached V is shared by every caller: V should be a value type or treated
// as read-only.
type Cache[V any] struct {
	cfg    Config
	now    func() time.Time
	flight singleflight.Group

	mu    sync.Mutex
	order *list.List // of *entry[V], most recently used first
	items map[string]*list.Element
}

// entry is one cached answer: a value, or an error accepted by
// Config.Negative.
type entry[V any] struct {
	key     string
	value   V
	err     error
	expires time.Time
}

// New returns an empty Cache, or ErrInvalidConfig.
func New[V any](cfg Config) (*Cache[V], error) {
	cfg, err := cfg.withDefaults()
	if err != nil {
		return nil, err
	}
	return &Cache[V]{
		cfg:   cfg,
		now:   time.Now,
		order: list.New(),
		items: make(map[string]*list.Element),
	}, nil
}

// Get returns the answer cached for key, or calls load once for all the
// callers that miss on key at the same time and caches its answer.
//
// A caller whose ctx ends stops waiting and gets ctx.Err(), but the load
// goes on for the others with a context that is never cancelled: load must
// bound its own duration, as the platform HTTP client does.
func (c *Cache[V]) Get(ctx context.Context, key string, load func(context.Context) (V, error)) (V, error) {
	if e, ok := c.lookup(key); ok {
		return e.value, e.err
	}
	loadCtx := context.WithoutCancel(ctx)
	results := c.flight.DoChan(key, func() (any, error) {
		return c.fill(loadCtx, key, load)
	})
	select {
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	case r := <-results:
		v, _ := r.Val.(V) // a nil interface V does not assert
		return v, r.Err
	}
}

// fill loads key unless a load that just ended cached it, then caches the
// answer.
func (c *Cache[V]) fill(ctx context.Context, key string, load func(context.Context) (V, error)) (V, error) {
	if e, ok := c.lookup(key); ok {
		return e.value, e.err
	}
	v, err := load(ctx)
	c.store(key, v, err)
	return v, err
}
