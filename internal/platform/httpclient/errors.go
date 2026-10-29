// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

var (
	// ErrURLNotAllowed means a URL, or a redirect target, is not HTTPS on an
	// allowed host, or carries credentials. No request was sent to it.
	ErrURLNotAllowed = errors.New("URL not allowed")

	// ErrBodyTooLarge means a response body exceeds Config.MaxBodyBytes.
	ErrBodyTooLarge = errors.New("response body too large")

	// ErrInvalidConfig means New was given a Config it cannot use.
	ErrInvalidConfig = errors.New("invalid HTTP client config")
)

// StatusError is a response whose status is not 2xx, returned once retries,
// if any, are over. Match it with errors.As to read StatusCode, for example
// to read a 404 from a registry as "not found".
type StatusError struct {
	// StatusCode is the HTTP status of the last response.
	StatusCode int

	retryAfter    time.Duration
	hasRetryAfter bool
}

// Error returns the status, such as "unexpected HTTP status 404 Not Found".
func (e *StatusError) Error() string {
	return "unexpected HTTP status " + strconv.Itoa(e.StatusCode) + " " + http.StatusText(e.StatusCode)
}
