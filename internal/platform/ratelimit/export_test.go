// SPDX-License-Identifier: AGPL-3.0-only

package ratelimit

// Len returns the number of keys l keeps a bucket for.
func Len(l *Limiter) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
