// SPDX-License-Identifier: AGPL-3.0-only

package cache

import "time"

// SetNow replaces the clock that dates and expires entries.
func (c *Cache[V]) SetNow(now func() time.Time) {
	c.now = now
}
