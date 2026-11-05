// SPDX-License-Identifier: AGPL-3.0-only

package objectstore_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/app"
)

var _ app.KeyDumpFetcher = (*objectstore.PresignedObject)(nil)

// signature is the query of a presigned URL; its value is the credential.
const signature = "?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Expires=3600&X-Amz-Signature=0123abcd"

func TestParsePresignedAcceptsAnObjectOfAmazonS3InParis(t *testing.T) {
	t.Parallel()
	tests := map[string]struct{ url, host string }{
		"virtual-hosted": {"https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com/restore/keys.tar.age" + signature, "arnakoff-key-backups.s3.eu-west-3.amazonaws.com"},
		"path style":     {"https://s3.eu-west-3.amazonaws.com/arnakoff-key-backups/restore/keys.tar.age" + signature, "s3.eu-west-3.amazonaws.com"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			u, err := objectstore.ParsePresigned(tt.url)
			if err != nil {
				t.Fatalf("ParsePresigned() error = %v", err)
			}
			if u.Host() != tt.host {
				t.Errorf("Host() = %q, want %q", u.Host(), tt.host)
			}
		})
	}
}

func TestParsePresignedRefuses(t *testing.T) {
	t.Parallel()
	const key = "/restore/keys.tar.age"
	tests := map[string]string{
		"empty":             "",
		"an s3 URI":         "s3://arnakoff-key-backups" + key,
		"plain HTTP":        "http://arnakoff-key-backups.s3.eu-west-3.amazonaws.com" + key + signature,
		"another region":    "https://arnakoff-key-backups.s3.eu-west-1.amazonaws.com" + key + signature,
		"the global host":   "https://arnakoff-key-backups.s3.amazonaws.com" + key + signature,
		"a lookalike host":  "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com.example.org" + key + signature,
		"a nested host":     "https://a.b.s3.eu-west-3.amazonaws.com" + key + signature,
		"a port":            "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com:8443" + key + signature,
		"credentials":       "https://user:pass@arnakoff-key-backups.s3.eu-west-3.amazonaws.com" + key + signature,
		"no signature":      "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com" + key,
		"no object":         "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com/" + signature,
		"path style bucket": "https://s3.eu-west-3.amazonaws.com/arnakoff-key-backups/" + signature,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := objectstore.ParsePresigned(raw)
			if !errors.Is(err, objectstore.ErrPresignedURL) {
				t.Fatalf("ParsePresigned() error = %v, want ErrPresignedURL", err)
			}
			if strings.Contains(err.Error(), "0123abcd") || strings.Contains(err.Error(), "pass") {
				t.Errorf("error %q quotes the URL's credential", err)
			}
		})
	}
}

// getterFake answers body or err, and keeps the URL it was asked.
type getterFake struct {
	body  []byte
	err   error
	asked string
}

func (g *getterFake) Get(_ context.Context, rawURL string) ([]byte, error) {
	g.asked = rawURL
	return g.body, g.err
}

func TestPresignedObjectFetchesItsURL(t *testing.T) {
	t.Parallel()
	const raw = "https://arnakoff-key-backups.s3.eu-west-3.amazonaws.com/restore/keys.tar.age" + signature
	u, err := objectstore.ParsePresigned(raw)
	if err != nil {
		t.Fatal(err)
	}
	g := &getterFake{body: []byte("sealed")}
	got, err := objectstore.NewPresignedObject(u, g).FetchDump(t.Context())
	if err != nil || string(got) != "sealed" || g.asked != raw {
		t.Errorf("FetchDump() = %q, %v after asking %q; want %q, nil after asking %q", got, err, g.asked, "sealed", raw)
	}

	failed := errors.New("unexpected HTTP status 403 Forbidden")
	if _, err := objectstore.NewPresignedObject(u, &getterFake{err: failed}).FetchDump(t.Context()); !errors.Is(err, failed) {
		t.Errorf("FetchDump() error = %v, want %v", err, failed)
	}
}

func TestPresignedURLPrintsWithoutItsQuery(t *testing.T) {
	t.Parallel()
	u, err := objectstore.ParsePresigned("https://s3.eu-west-3.amazonaws.com/arnakoff-key-backups/restore/keys.tar.age" + signature)
	if err != nil {
		t.Fatal(err)
	}
	const want = "https://s3.eu-west-3.amazonaws.com/arnakoff-key-backups/restore/keys.tar.age"
	for _, got := range []string{fmt.Sprint(u), fmt.Sprintf("%+v", u), fmt.Sprintf("%#v", u)} {
		if got != want {
			t.Errorf("printed %q, want %q", got, want)
		}
	}
}
