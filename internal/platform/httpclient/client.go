// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/arnakoff/api/internal/platform/clock"
)

// maxRedirects matches the limit of net/http's default policy.
const maxRedirects = 10

// drainBytes is how much of an error body is read so the connection can be
// reused.
const drainBytes = 4 << 10

// Client is an outbound HTTP client restricted to allowed hosts. It is safe
// for concurrent use.
type Client struct {
	http      *http.Client
	hosts     allowlist
	userAgent string
	timeout   time.Duration
	maxBody   int64
	retry     retryPolicy
	wait      func(ctx context.Context, d time.Duration) error
	now       func() time.Time
}

// New returns a Client for cfg, or ErrInvalidConfig.
func New(cfg Config) (*Client, error) {
	cfg, err := cfg.withDefaults()
	if err != nil {
		return nil, err
	}
	hosts, err := newAllowlist(cfg.AllowedHosts)
	if err != nil {
		return nil, err
	}
	c := &Client{
		hosts:     hosts,
		userAgent: cfg.UserAgent,
		timeout:   cfg.Timeout,
		maxBody:   cfg.MaxBodyBytes,
		retry:     retryPolicy{attempts: cfg.Attempts, base: cfg.BaseBackoff, max: cfg.MaxBackoff},
		wait:      clock.System{}.Sleep,
		now:       clock.System{}.Now,
	}
	c.http = &http.Client{Transport: newTransport(cfg), CheckRedirect: c.checkRedirect}
	return c, nil
}

func newTransport(cfg Config) *http.Transport {
	return &http.Transport{
		// No proxy, not even from the environment: see the package doc.
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: cfg.DialTimeout}).DialContext,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: cfg.RootCAs},
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	}
}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	if !c.hosts.allows(req.URL) {
		return fmt.Errorf("redirect: %w", ErrURLNotAllowed)
	}
	return nil
}

// Get fetches rawURL and returns its body. It returns ErrURLNotAllowed
// without sending anything when rawURL is not HTTPS on an allowed host,
// ErrBodyTooLarge, a *StatusError for a non-2xx answer, or the error of the
// transport or of ctx.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := c.parse(rawURL)
	if err != nil {
		return nil, err
	}
	body, err := c.send(ctx, request{method: http.MethodGet, url: u})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", u, err)
	}
	return body, nil
}

// Post sends body to rawURL with the given content type and discards the
// answer. It is retried like Get, so the receiver may get body twice. Its
// errors, those of Get otherwise, never quote rawURL: a webhook URL carries
// a secret.
func (c *Client) Post(ctx context.Context, rawURL, contentType string, body []byte) error {
	u, err := c.parse(rawURL)
	if err != nil {
		return err
	}
	if _, err := c.send(ctx, request{method: http.MethodPost, url: u, contentType: contentType, body: body}); err != nil {
		return fmt.Errorf("post: %w", err)
	}
	return nil
}

// parse returns rawURL parsed, or ErrURLNotAllowed.
func (c *Client) parse(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || !c.hosts.allows(u) {
		// The URL is left out: it may carry credentials.
		return nil, ErrURLNotAllowed
	}
	return u, nil
}

// request is what Get and Post send, built anew at each attempt.
type request struct {
	method      string
	url         *url.URL
	contentType string
	body        []byte
}

// send tries r until an answer is final and returns its body. Its errors
// do not name the URL.
func (c *Client) send(ctx context.Context, r request) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	for attempt := 1; ; attempt++ {
		body, err := c.attempt(ctx, r)
		delay, ok := c.retry.next(attempt, err)
		if !ok {
			return body, err
		}
		if err := c.wait(ctx, delay); err != nil {
			return nil, fmt.Errorf("wait before retry: %w", err)
		}
	}
}

func (c *Client) attempt(ctx context.Context, r request) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, r.method, r.url.String(), bytes.NewReader(r.body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, withoutURL(err)
	}
	defer func() { _ = resp.Body.Close() }() // The body is read in full; a close error changes nothing.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, drainBytes))
		return nil, c.statusError(resp)
	}
	return readBody(resp, c.maxBody)
}

// withoutURL drops the URL a *url.Error quotes, keeping its cause.
func withoutURL(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err
	}
	return err
}

func (c *Client) statusError(resp *http.Response) *StatusError {
	e := &StatusError{StatusCode: resp.StatusCode}
	e.retryAfter, e.hasRetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
	return e
}

// readBody reads at most limit bytes, and ErrBodyTooLarge past them.
func readBody(resp *http.Response, limit int64) ([]byte, error) {
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("%w: %d bytes announced, limit %d", ErrBodyTooLarge, resp.ContentLength, limit)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%w: limit %d bytes", ErrBodyTooLarge, limit)
	}
	return body, nil
}
