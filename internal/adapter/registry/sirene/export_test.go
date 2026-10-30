// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import (
	"time"

	"golang.org/x/time/rate"

	"github.com/arnakoff/api/internal/platform/throttle"
)

// SetNow replaces the clock that dates the API's Retry-After.
func (c *Client) SetNow(now func() time.Time) {
	c.throttle = throttle.New(now)
}

// SetLimiter replaces the request rate limiter.
func (c *Client) SetLimiter(l *rate.Limiter) {
	c.limiter = l
}

// Limiter returns the request rate limiter.
func (c *Client) Limiter() *rate.Limiter {
	return c.limiter
}
