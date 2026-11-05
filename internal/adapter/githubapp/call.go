// SPDX-License-Identifier: AGPL-3.0-only

package githubapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Host is the host of the GitHub REST API, the only one the App client
// may reach.
const Host = "api.github.com"

// APIVersion is the version of the REST API the requests ask for, the one
// the documentation of the calls gives (docs.github.com, read on
// 05/11/2026).
const APIVersion = "2026-03-10"

// Sender sends a request: the platform client, which allows Host only,
// bounds time and response size, retries 429 and 5xx answers and returns
// any other non-2xx answer as a *httpclient.StatusError.
type Sender interface {
	Send(ctx context.Context, method, rawURL string, header http.Header, body []byte) ([]byte, error)
}

// request is a call of the REST API.
type request struct {
	method, path, bearer string
	body                 []byte
}

// call sends r with bearer as its token and decodes the answer into out,
// unless out is nil. Every call is safe to repeat: the token request
// mints another token, the public key is read again and a secret is
// written again with the same value. Its errors never quote the token nor
// the body sent.
func (s *EnvironmentSecrets) call(ctx context.Context, r request, out any) error {
	header := http.Header{
		"Accept":               {"application/vnd.github+json"},
		"X-GitHub-Api-Version": {APIVersion},
		"Authorization":        {"Bearer " + r.bearer},
	}
	if r.body != nil {
		header.Set("Content-Type", "application/json")
	}
	answer, err := s.opts.HTTP.Send(ctx, r.method, s.opts.APIURL+r.path, header, r.body)
	if err != nil {
		return fmt.Errorf("%s %s: %w", r.method, r.path, err)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(answer, out); err != nil {
		return fmt.Errorf("%s %s: decode the answer: %w", r.method, r.path, err)
	}
	return nil
}
