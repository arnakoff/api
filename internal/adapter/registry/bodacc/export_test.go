// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import (
	"time"

	"github.com/arnakoff/api/internal/platform/throttle"
)

// SetNow replaces the clock that dates the API's Retry-After.
func (c *Client) SetNow(now func() time.Time) {
	c.throttle = throttle.New(now)
}
