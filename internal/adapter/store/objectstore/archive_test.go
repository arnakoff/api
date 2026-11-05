// SPDX-License-Identifier: AGPL-3.0-only

package objectstore_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

var _ app.BackupArchive = (*objectstore.Archive)(nil)

// newArchive returns an Archive on a fake bucket server, and the server.
func newArchive(t *testing.T) (*objectstore.Archive, *fakeS3) {
	t.Helper()
	fake := newFakeS3(t)
	srv := httptest.NewTLSServer(fake)
	t.Cleanup(srv.Close)
	a, err := objectstore.NewArchive(objectstore.Options{
		Endpoint:  srv.URL,
		Region:    "eu-west-3",
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: config.Secret(secretKey),
		HTTP:      httpclienttest.NewClient(t, srv, httpclient.Config{}),
		Attempts:  1,
	})
	if err != nil {
		t.Fatalf("NewArchive() error = %v", err)
	}
	return a, fake
}

func TestArchiveKeepsANewObject(t *testing.T) {
	t.Parallel()
	a, fake := newArchive(t)
	if err := a.Keep(context.Background(), "keys/2026-11-05T023000Z.tar.age", []byte("sealed")); err != nil {
		t.Fatalf("Keep() error = %v", err)
	}
	got, ok := fake.object("/" + bucket + "/keys/2026-11-05T023000Z.tar.age")
	if !ok || string(got) != "sealed" {
		t.Errorf("object = %q, %v; want the sealed backup", got, ok)
	}
	// A backup never replaces another one (S3 conditional write).
	if inm := fake.lastIfNoneMatch(); inm != "*" {
		t.Errorf("If-None-Match = %q, want *", inm)
	}
}

func TestArchiveKeepFailsWhenTheBucketRefuses(t *testing.T) {
	t.Parallel()
	a, fake := newArchive(t)
	fake.fail(http.StatusForbidden)
	if err := a.Keep(context.Background(), "keys/x.tar.age", []byte("sealed")); err == nil {
		t.Error("Keep() error = nil, want the refusal")
	}
}

func TestNewArchiveRefusesIncompleteOptions(t *testing.T) {
	t.Parallel()
	_, err := objectstore.NewArchive(objectstore.Options{Bucket: bucket})
	if !errors.Is(err, objectstore.ErrInvalidOptions) {
		t.Errorf("NewArchive() error = %v, want ErrInvalidOptions", err)
	}
}
