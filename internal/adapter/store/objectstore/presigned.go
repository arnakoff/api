// SPDX-License-Identifier: AGPL-3.0-only

package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
)

// ErrPresignedURL means a URL is not a presigned URL of an object of Amazon
// S3 in Paris. The error never quotes the URL: its query is a credential.
var ErrPresignedURL = errors.New("not a presigned URL of an object of Amazon S3 in eu-west-3")

// bucketLabel is a bucket name that can prefix the host of a virtual-hosted
// URL: lower-case letters, digits and hyphens, 3 to 63 characters (Amazon S3
// User Guide, bucketnamingrules; a name with dots does not match the
// certificate of the host).
var bucketLabel = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

// PresignedURL is a presigned URL of an object of Amazon S3 in Paris, such
// as the one `aws s3 presign` prints, checked by ParsePresigned.
// Every fmt verb prints it without its query.
type PresignedURL struct {
	raw  string
	bare string
	host string
}

// ParsePresigned returns rawURL checked: HTTPS, without credentials nor
// port, on AWSParisHost (path style) or on a bucket of it (virtual-hosted),
// naming an object, with a SigV4 signature in its query. Otherwise it
// returns ErrPresignedURL.
func ParsePresigned(rawURL string) (PresignedURL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Query().Get("X-Amz-Signature") == "" {
		return PresignedURL{}, ErrPresignedURL
	}
	key := strings.TrimPrefix(u.EscapedPath(), "/")
	if u.Host == AWSParisHost {
		_, key, _ = strings.Cut(key, "/")
	} else if bucket, ok := strings.CutSuffix(u.Host, "."+AWSParisHost); !ok || !bucketLabel.MatchString(bucket) {
		return PresignedURL{}, ErrPresignedURL
	}
	if key == "" {
		return PresignedURL{}, ErrPresignedURL
	}
	bare := url.URL{Scheme: u.Scheme, Host: u.Host, Path: u.Path, RawPath: u.RawPath}
	return PresignedURL{raw: rawURL, bare: bare.String(), host: u.Host}, nil
}

// Host returns the host of the URL, the one host its client may reach.
func (u PresignedURL) Host() string { return u.host }

// Format implements fmt.Formatter: every verb writes the URL without its
// query, which is a credential.
func (u PresignedURL) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, u.bare)
}

// getter fetches a URL, as httpclient.Client.Get does.
type getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// PresignedObject reads one object by its presigned URL, with no
// credentials of its own (#297). It implements app.KeyDumpFetcher.
type PresignedObject struct {
	url    PresignedURL
	client getter
}

// NewPresignedObject returns the object of u, fetched with client, which
// must allow u.Host() only and never quote the query of a URL in its
// errors.
func NewPresignedObject(u PresignedURL, client getter) *PresignedObject {
	return &PresignedObject{url: u, client: client}
}

// FetchDump returns the body of the object.
func (o *PresignedObject) FetchDump(ctx context.Context) ([]byte, error) {
	return o.client.Get(ctx, o.url.raw)
}
