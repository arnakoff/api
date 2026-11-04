// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"testing"
	"time"
)

// BenchmarkKeyCacheHit is the lookup of a valid key on the /check hot path.
func BenchmarkKeyCacheHit(b *testing.B) {
	cache, _, _, _, d := newCachedKeys(&testing.T{}, 30*time.Second)
	ctx := context.Background()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := cache.FindKey(ctx, d); err != nil {
				b.Fatal(err)
			}
		}
	})
}
