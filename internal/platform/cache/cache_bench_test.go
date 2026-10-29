// SPDX-License-Identifier: AGPL-3.0-only

package cache_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/arnakoff/api/internal/platform/cache"
)

func loadKey(context.Context) (string, error) { return "v", nil }

func newBenchCache(b *testing.B, maxEntries int) *cache.Cache[string] {
	b.Helper()
	c, err := cache.New[string](cache.Config{MaxEntries: maxEntries})
	if err != nil {
		b.Fatal(err)
	}
	return c
}

func BenchmarkGetHit(b *testing.B) {
	c := newBenchCache(b, 0)
	ctx := b.Context()
	_, _ = c.Get(ctx, "k", loadKey)
	for b.Loop() {
		_, _ = c.Get(ctx, "k", loadKey)
	}
}

func BenchmarkGetHitParallel(b *testing.B) {
	c := newBenchCache(b, 0)
	ctx := b.Context()
	_, _ = c.Get(ctx, "k", loadKey)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Get(ctx, "k", loadKey)
		}
	})
}

// BenchmarkGetMissEvict loads a new key at each call into a full cache, so
// every call loads, stores and evicts.
func BenchmarkGetMissEvict(b *testing.B) {
	const size = 1024
	c := newBenchCache(b, size)
	ctx := b.Context()
	keys := make([]string, 4*size)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	i := 0
	for b.Loop() {
		_, _ = c.Get(ctx, keys[i%len(keys)], loadKey)
		i++
	}
}
