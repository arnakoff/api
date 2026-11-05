// SPDX-License-Identifier: AGPL-3.0-only

package objectstore_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// Compile-time check: KeyCheck checks a new key before it is in service.
var _ app.CredentialInstaller = (*objectstore.KeyCheck)(nil)

var rotated = app.Credential{AccessKey: "SCWNEWKEYXXXXXXXXXXX", SecretKey: []byte("33333333-3333-3333-3333-333333333333")}

// newKeyCheck returns a KeyCheck on a fake bucket server, the server and
// the pauses it made.
func newKeyCheck(t *testing.T) (*objectstore.KeyCheck, *fakeS3, *[]time.Duration) {
	t.Helper()
	fake := newFakeS3(t)
	srv := httptest.NewTLSServer(fake)
	t.Cleanup(srv.Close)
	var pauses []time.Duration
	check, err := objectstore.NewKeyCheck(objectstore.KeyCheckOptions{
		Endpoint: srv.URL,
		Bucket:   bucket,
		HTTP:     httpclienttest.NewClient(t, srv, httpclient.Config{}),
		Attempts: 3,
		Pause:    10 * time.Second,
		Sleep: func(_ context.Context, d time.Duration) error {
			pauses = append(pauses, d)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return check, fake, &pauses
}

func TestKeyCheckListsTheBucketWithTheNewKey(t *testing.T) {
	t.Parallel()
	check, fake, pauses := newKeyCheck(t)

	if err := check.InstallCredential(context.Background(), rotated); err != nil {
		t.Fatal(err)
	}

	if fake.listed() != 1 || len(*pauses) != 0 {
		t.Errorf("%d listings, %d pauses, want one listing at once", fake.listed(), len(*pauses))
	}
	if !strings.Contains(fake.auth(), "Credential="+rotated.AccessKey+"/") {
		t.Errorf("signed with %q, want the new key", fake.auth())
	}
}

func TestKeyCheckWaitsForTheNewKeyToWork(t *testing.T) {
	t.Parallel()
	check, fake, pauses := newKeyCheck(t)
	fake.failFor(http.StatusForbidden, 2)

	if err := check.InstallCredential(context.Background(), rotated); err != nil {
		t.Fatal(err)
	}

	if len(*pauses) != 2 || (*pauses)[0] != 10*time.Second {
		t.Errorf("pauses %v, want two of 10s", *pauses)
	}
}

func TestKeyCheckFailsWhenTheKeyNeverWorks(t *testing.T) {
	t.Parallel()
	check, fake, pauses := newKeyCheck(t)
	fake.fail(http.StatusForbidden)

	err := check.InstallCredential(context.Background(), rotated)

	if err == nil || !strings.Contains(err.Error(), rotated.AccessKey) {
		t.Errorf("err = %v, want an error naming the key", err)
	}
	if strings.Contains(err.Error(), string(rotated.SecretKey)) {
		t.Error("error quotes the secret key")
	}
	if len(*pauses) != 2 {
		t.Errorf("%d pauses, want 2 between 3 attempts", len(*pauses))
	}
}

func TestKeyCheckStopsWhenTheContextIsDone(t *testing.T) {
	t.Parallel()
	check, fake, _ := newKeyCheck(t)
	fake.fail(http.StatusForbidden)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := check.InstallCredential(ctx, rotated); err == nil {
		t.Error("no error once the context is done")
	}
}

func TestNewKeyCheckRefusesBadOptions(t *testing.T) {
	t.Parallel()
	client := &http.Client{}
	cases := map[string]objectstore.KeyCheckOptions{
		"no bucket":         {HTTP: client},
		"no HTTP client":    {Bucket: bucket},
		"negative attempts": {Bucket: bucket, HTTP: client, Attempts: -1},
		"negative pause":    {Bucket: bucket, HTTP: client, Pause: -time.Second},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := objectstore.NewKeyCheck(opts); !errors.Is(err, objectstore.ErrInvalidOptions) {
				t.Errorf("NewKeyCheck() = %v, want ErrInvalidOptions", err)
			}
		})
	}
}
