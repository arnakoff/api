// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"context"
	"time"
)

// SetWait replaces the pause between two attempts, so tests never sleep.
func (c *Client) SetWait(wait func(ctx context.Context, d time.Duration) error) {
	c.wait = wait
}

// SetNow replaces the clock used to read a Retry-After date.
func (c *Client) SetNow(now func() time.Time) {
	c.now = now
}
