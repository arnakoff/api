// SPDX-License-Identifier: AGPL-3.0-only

package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/platform/httpclient"
)

// siteverifyHost serves the siteverify API.
const siteverifyHost = "challenges.cloudflare.com"

// SiteverifyURL is the endpoint of the siteverify API.
const SiteverifyURL = "https://" + siteverifyHost + "/turnstile/v0/siteverify"

// Hosts returns the hosts a Verifier reaches.
func Hosts() []string {
	return []string{siteverifyHost}
}

// maxTokenLen is the longest token siteverify accepts.
const maxTokenLen = 2048

// The bounds of a call to siteverify: a demo check waits for it.
const (
	timeout      = 5 * time.Second
	maxBodyBytes = 64 << 10
)

// refusalCodes are the error codes of a token siteverify refuses: absent,
// invalid, expired or already used. Any other code means the token could
// not be checked.
var refusalCodes = []string{"missing-input-response", "invalid-input-response", "timeout-or-duplicate"}

// errUnchecked means siteverify did not tell whether the token is valid.
var errUnchecked = errors.New("turnstile token not checked")

// Submitter posts a body and returns the answer; *httpclient.Client is
// one. Its errors must not quote the body.
type Submitter interface {
	Submit(ctx context.Context, rawURL, contentType string, body []byte) ([]byte, error)
}

// Verifier checks the tokens of one Turnstile widget, served on one
// hostname.
type Verifier struct {
	client   Submitter
	secret   []byte
	hostname string
}

// New returns a Verifier posting through client the secret key of the
// widget, whose valid tokens name hostname.
func New(client Submitter, secret []byte, hostname string) *Verifier {
	return &Verifier{client: client, secret: secret, hostname: hostname}
}

// NewLive returns a Verifier reaching siteverify through an HTTP client
// allowed on its host only, trying each token once.
func NewLive(secret []byte, hostname string) (*Verifier, error) {
	client, err := httpclient.New(httpclient.Config{
		AllowedHosts: Hosts(), Timeout: timeout, MaxBodyBytes: maxBodyBytes, Attempts: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("turnstile: %w", err)
	}
	return New(client, secret, hostname), nil
}

// answer is the part of a siteverify answer Verify reads.
type answer struct {
	Success    bool     `json:"success"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify reports whether token is a valid token of the widget, solved on
// its hostname. A refused token is false with a nil error; an error means
// the token could not be checked. The visitor's address is not sent: it is
// a personal datum, and siteverify does not require it.
func (v *Verifier) Verify(ctx context.Context, token string) (bool, error) {
	if token == "" || len(token) > maxTokenLen {
		return false, nil
	}
	body, err := v.client.Submit(ctx, SiteverifyURL, "application/x-www-form-urlencoded", v.form(token))
	if err != nil {
		return false, fmt.Errorf("siteverify: %w", err)
	}
	var a answer
	if err := json.Unmarshal(body, &a); err != nil {
		return false, fmt.Errorf("siteverify answer: %w", errUnchecked)
	}
	return a.verdict(v.hostname)
}

// form returns the body of the siteverify request of token.
func (v *Verifier) form(token string) []byte {
	return []byte(url.Values{"secret": {string(v.secret)}, "response": {token}}.Encode())
}

// verdict tells whether a admits a token solved on hostname. A failure is
// a refusal only when every error code refuses the token.
func (a answer) verdict(hostname string) (bool, error) {
	if a.Success {
		return a.Hostname == hostname, nil
	}
	unchecked := fmt.Errorf("siteverify error codes %q: %w", a.ErrorCodes, errUnchecked)
	if len(a.ErrorCodes) == 0 {
		return false, unchecked
	}
	for _, code := range a.ErrorCodes {
		if !slices.Contains(refusalCodes, code) {
			return false, unchecked
		}
	}
	return false, nil
}
