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
// Config.Timeout bounds the whole exchange, body included; reading more
// than Config.MaxBodyBytes of the body fails with ErrBodyTooLarge. It does
// not retry and leaves the User-Agent of req. The caller closes the body.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if !c.hosts.allows(req.URL) {
		return nil, ErrURLNotAllowed
	}
	//nolint:gosec // G704: the host is checked against the allowlist above; redirects by checkRedirect.
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
