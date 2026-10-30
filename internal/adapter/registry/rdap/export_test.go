// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"time"

	"github.com/arnakoff/api/internal/platform/throttle"
)

// SetNow replaces the clock that dates a registry's Retry-After.
func (c *Client) SetNow(now func() time.Time) {
	c.throttle = throttle.New(now)
}
