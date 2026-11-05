// SPDX-License-Identifier: AGPL-3.0-only

package scaleway

import (
	"errors"
	"net/http"

	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/arnakoff/api/internal/platform/config"
)

// Host is the host of the Scaleway API, the only one the client of the
// rotator may reach (ADR 0007).
const Host = "api.scaleway.com"

// region is where the secrets and the containers of production are
// (ADR 0005).
const region = scw.RegionFrPar

// ErrInvalidOptions means NewClient was given Options it cannot use. Its
// errors never quote a key.
var ErrInvalidOptions = errors.New("invalid Scaleway client options")

// Doer sends a request: the platform client, which allows Host only and
// bounds time and response size.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Options configures the Scaleway client.
type Options struct {
	// HTTP sends the requests. Required.
	HTTP Doer

	// APIURL is the base URL of the API; "https://" + Host when empty.
	APIURL string

	// AccessKey and SecretKey are the key of the rotator. Required.
	AccessKey string
	SecretKey config.Secret
}

// NewClient returns a Scaleway client of opts. It reads no environment
// variable nor configuration file: every setting comes from opts. The
// request dumper the SDK turns on with SCW_DEBUG wraps a bare *http.Client
// only (scw/client.go, setRequestLogging), never opts.HTTP, so no request
// nor answer, a new secret key included, reaches a log.
func NewClient(opts Options) (*scw.Client, error) {
	if opts.HTTP == nil {
		return nil, errors.Join(ErrInvalidOptions, errors.New("no HTTP client"))
	}
	if opts.APIURL == "" {
		opts.APIURL = "https://" + Host
	}
	c, err := scw.NewClient(
		scw.WithHTTPClient(opts.HTTP),
		scw.WithAPIURL(opts.APIURL),
		scw.WithAuth(opts.AccessKey, string(opts.SecretKey.Bytes())),
		scw.WithDefaultRegion(region),
	)
	if err != nil {
		// The SDK quotes a malformed key in its error: leave it out.
		return nil, errors.Join(ErrInvalidOptions, errors.New("an access key SCW… and a secret key UUID are required"))
	}
	return c, nil
}
