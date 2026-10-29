// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"errors"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// retryPolicy decides whether and when to try again.
type retryPolicy struct {
	attempts int
	base     time.Duration
	max      time.Duration
}

// next returns the pause before the attempt that follows attempt (counted
// from 1), and false when err is final: no error, an error other than a 429
// or 5xx status, the last attempt, or a Retry-After longer than max.
func (p retryPolicy) next(attempt int, err error) (time.Duration, bool) {
	var se *StatusError
	if !errors.As(err, &se) || !retryable(se.StatusCode) || attempt >= p.attempts {
		return 0, false
	}
	if se.hasRetryAfter {
		return se.retryAfter, se.retryAfter <= p.max
	}
	return p.backoff(attempt), true
}

func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// backoff draws the pause after attempt (counted from 1) from
// [0, min(base·2^(attempt-1), max)]: exponential backoff with full jitter.
func (p retryPolicy) backoff(attempt int) time.Duration {
	ceiling := p.base
	for i := 1; i < attempt && ceiling < p.max; i++ {
		ceiling *= 2
	}
	ceiling = min(ceiling, p.max)
	//nolint:gosec // G404: jitter spreads retries, it needs no cryptographic randomness.
	return time.Duration(rand.Int64N(int64(ceiling) + 1))
}

// parseRetryAfter reads a Retry-After value (RFC 9110, section 10.2.3):
// delay-seconds or an HTTP-date, relative to now. A date in the past is no
// wait; an invalid value is no Retry-After.
func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.ParseInt(v, 10, 64); err == nil || errors.Is(err, strconv.ErrRange) {
		switch {
		case secs < 0:
			return 0, false
		case secs > int64(math.MaxInt64/time.Second):
			return time.Duration(math.MaxInt64), true
		}
		return time.Duration(secs) * time.Second, true
	}
	t, err := http.ParseTime(v)
	if err != nil {
		return 0, false
	}
	return max(t.Sub(now), 0), true
}
