// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"fmt"
	"io"
	"net/http"
)

// Do sends req as built and returns the response, whatever its status, for
// a library that builds its own requests, such as an S3 SDK. It keeps the
// rules of Get: req must be HTTPS on an allowed host, or ErrURLNotAllowed
// is returned without sending anything; a redirect elsewhere is refused;
// Config.Timeout bounds each exchange, body included; reading more than
// Config.MaxBodyBytes of the body fails with ErrBodyTooLarge. It leaves
// the User-Agent of req. The caller closes the body.
//
// It does not retry unless Config.RetryDo is set. Then a 429 or 5xx answer
// to an idempotent request (RFC 9110, section 9.2.2) whose body can be sent
// again (none, or http.Request.GetBody) is retried as Get does, the pauses
// bounded by the context of req; the last answer is returned as is. A POST
// or a PATCH is sent once: a retry could repeat what the first one did.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if !c.hosts.allows(req.URL) {
		return nil, ErrURLNotAllowed
	}
	for attempt := 1; ; attempt++ {
		resp, err := c.doOnce(req)
		if err != nil {
			return nil, err
		}
		delay, ok := c.retryDoAfter(req, resp, attempt)
		if !ok {
			return resp, nil
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, drainBytes))
		_ = resp.Body.Close() // The answer is dropped; a close error changes nothing.
		if err := c.wait(req.Context(), delay); err != nil {
			return nil, fmt.Errorf("wait before retry: %w", err)
		}
		if req, err = rewind(req); err != nil {
			return nil, err
		}
	}
}

// doOnce sends req once and caps the body of the answer.
func (c *Client) doOnce(req *http.Request) (*http.Response, error) {
	//nolint:gosec // G704: Do checks the host against the allowlist; redirects by checkRedirect.
	resp, err := c.raw.Do(req)
	if err != nil {
		return nil, withoutURL(err)
	}
	if resp.ContentLength > c.maxBody {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: %d bytes announced, limit %d", ErrBodyTooLarge, resp.ContentLength, c.maxBody)
	}
	resp.Body = &cappedBody{ReadCloser: resp.Body, left: c.maxBody}
	return resp, nil
}

// cappedBody fails with ErrBodyTooLarge once more than left bytes are read.
type cappedBody struct {
	io.ReadCloser
	left int64
}

// Read implements io.Reader. It reads one byte past the limit to tell a
// body that ends at it from one that goes on.
func (b *cappedBody) Read(p []byte) (int, error) {
	if int64(len(p)) > b.left+1 {
		p = p[:b.left+1]
	}
	n, err := b.ReadCloser.Read(p)
	b.left -= int64(n)
	if b.left < 0 {
		return n + int(b.left), fmt.Errorf("%w: limit reached", ErrBodyTooLarge)
	}
	return n, err
}
