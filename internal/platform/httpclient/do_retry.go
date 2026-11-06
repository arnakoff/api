// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"fmt"
	"net/http"
	"time"
)

// retryDoAfter returns the pause before Do sends req again after resp,
// the answer to attempt (counted from 1), and false when resp is final.
func (c *Client) retryDoAfter(req *http.Request, resp *http.Response, attempt int) (time.Duration, bool) {
	if !c.retryDo || resp.StatusCode < 400 || !replayable(req) {
		return 0, false
	}
	return c.retry.next(attempt, c.statusError(resp))
}

// replayable reports whether req may be sent again: an idempotent method
// (RFC 9110, section 9.2.2) and a body that can be read anew.
func replayable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

// rewind returns a copy of req with its body read anew.
func rewind(req *http.Request) (*http.Request, error) {
	next := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return next, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("rewind the request body: %w", err)
	}
	next.Body = body
	return next, nil
}
