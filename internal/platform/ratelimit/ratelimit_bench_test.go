// SPDX-License-Identifier: AGPL-3.0-only

package ratelimit_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/platform/ratelimit"
)

// benchKeys is the number of clients the benchmarks spread requests over.
const benchKeys = 1024

func benchLimiter() *ratelimit.Limiter {
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	return ratelimit.New(ratelimit.Policy{Quota: 60, Window: time.Minute}, benchKeys, func() time.Time { return now })
}

func benchKeyNames() []string {
	keys := make([]string, benchKeys)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	return keys
}

func BenchmarkAllow(b *testing.B) {
	l, keys := benchLimiter(), benchKeyNames()
	i := 0
	for b.Loop() {
		l.Allow(keys[i%benchKeys])
		i++
	}
}

func BenchmarkAllowParallel(b *testing.B) {
	l, keys := benchLimiter(), benchKeyNames()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			l.Allow(keys[i%benchKeys])
			i++
		}
	})
}
