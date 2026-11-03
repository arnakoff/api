// SPDX-License-Identifier: AGPL-3.0-only

package objectstore_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/adapter/store/objectstore"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

const (
	bucket    = "arnakoff-snapshots"
	accessKey = "SCWTESTACCESSKEY"
	secretKey = "test-secret-key"
)

// Compile-time check: Store is a snapshot store.
var _ app.SnapshotStore = (*objectstore.Store)(nil)

// newStore returns a Store on a fake bucket server, and the server.
func newStore(t *testing.T) (*objectstore.Store, *fakeS3) {
	t.Helper()
	fake := newFakeS3(t)
	srv := httptest.NewTLSServer(fake)
	t.Cleanup(srv.Close)
	st, err := objectstore.New(objectstore.Options{
		Endpoint:  srv.URL,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: config.Secret(secretKey),
		HTTP:      httpclienttest.NewClient(t, srv, httpclient.Config{}),
		Attempts:  1,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return st, fake
}

func snapshot(fetched time.Time, rows ...[]string) domain.Snapshot {
	return domain.Snapshot{
		Source:    "AMF",
		Header:    []string{"nom", "site"},
		Rows:      rows,
		FetchedAt: fetched,
	}
}

var day1 = time.Date(2026, 11, 3, 6, 0, 0, 0, time.UTC)

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Parallel()
	st, fake := newStore(t)
	want := snapshot(day1, []string{"Example", "example.com"})

	if err := st.Save(t.Context(), want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := st.Load(t.Context(), "AMF")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
	stored, ok := fake.object("/" + bucket + "/amf.snapshot")
	if !ok {
		t.Fatalf("no object at /%s/amf.snapshot: want a path-style key named as the file", bucket)
	}
	if enc, _ := store.Encode(want); string(stored) != string(enc) {
		t.Errorf("stored object is not the snapshot file format")
	}
}

func TestRequestsAreSignedForTheRegion(t *testing.T) {
	t.Parallel()
	st, fake := newStore(t)
	_, _ = st.Load(t.Context(), "AMF")

	auth := fake.auth()
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential="+accessKey+"/") || !strings.Contains(auth, "/fr-par/s3/aws4_request") {
		t.Errorf("Authorization = %q, want SigV4 with the access key for fr-par", auth)
	}
	if strings.Contains(auth, secretKey) {
		t.Errorf("Authorization carries the secret key")
	}
}

func TestLoadWithoutObjectIsNoSnapshot(t *testing.T) {
	t.Parallel()
	st, _ := newStore(t)

	_, err := st.Load(t.Context(), "AMF")
	if !errors.Is(err, app.ErrNoSnapshot) {
		t.Errorf("Load() error = %v, want app.ErrNoSnapshot", err)
	}
}

func TestLoadDownloadsOnlyWhatChanged(t *testing.T) {
	t.Parallel()
	st, fake := newStore(t)
	first := snapshot(day1, []string{"A", "a.example"})
	if err := st.Save(t.Context(), first); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		got, err := st.Load(t.Context(), "AMF")
		if err != nil || !reflect.DeepEqual(got, first) {
			t.Fatalf("Load() = %+v, %v, want the first snapshot", got, err)
		}
	}
	if d, nm := fake.counts(); d != 1 || nm != 1 {
		t.Errorf("downloads, not modified = %d, %d, want 1, 1", d, nm)
	}

	second := snapshot(day1.Add(24*time.Hour), []string{"B", "b.example"})
	enc, _ := store.Encode(second)
	fake.put("/"+bucket+"/amf.snapshot", enc)
	got, err := st.Load(t.Context(), "AMF")
	if err != nil || !reflect.DeepEqual(got, second) {
		t.Errorf("Load() after a change = %+v, %v, want the second snapshot", got, err)
	}
}

func TestLoadRejectsACorruptObject(t *testing.T) {
	t.Parallel()
	st, fake := newStore(t)
	fake.put("/"+bucket+"/amf.snapshot", []byte("not a snapshot"))

	for range 2 {
		if _, err := st.Load(t.Context(), "AMF"); !errors.Is(err, store.ErrCorrupt) {
			t.Fatalf("Load() error = %v, want store.ErrCorrupt", err)
		}
	}
	if d, _ := fake.counts(); d != 2 {
		t.Errorf("downloads = %d, want 2: a rejected object is not remembered", d)
	}
}

func TestLoadForgetsADeletedObject(t *testing.T) {
	t.Parallel()
	st, fake := newStore(t)
	if err := st.Save(t.Context(), snapshot(day1)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load(t.Context(), "AMF"); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	delete(fake.objects, "/"+bucket+"/amf.snapshot")
	fake.mu.Unlock()

	if _, err := st.Load(t.Context(), "AMF"); !errors.Is(err, app.ErrNoSnapshot) {
		t.Errorf("Load() error = %v, want app.ErrNoSnapshot", err)
	}
}

func TestStoreReportsAFailingBucket(t *testing.T) {
	t.Parallel()
	st, fake := newStore(t)
	fake.fail(http.StatusInternalServerError)

	if err := st.Save(t.Context(), snapshot(day1)); err == nil {
		t.Errorf("Save() error = nil, want the failure")
	}
	_, err := st.Load(t.Context(), "AMF")
	if err == nil || errors.Is(err, app.ErrNoSnapshot) {
		t.Errorf("Load() error = %v, want a failure other than no snapshot", err)
	}
	if strings.Contains(err.Error(), secretKey) {
		t.Errorf("error %q quotes the secret key", err)
	}
}

func TestStoreHonoursTheContext(t *testing.T) {
	t.Parallel()
	st, _ := newStore(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := st.Save(ctx, snapshot(day1)); !errors.Is(err, context.Canceled) {
		t.Errorf("Save() error = %v, want context.Canceled", err)
	}
	if _, err := st.Load(ctx, "AMF"); !errors.Is(err, context.Canceled) {
		t.Errorf("Load() error = %v, want context.Canceled", err)
	}
}

func TestStoreRejectsASourceWithoutName(t *testing.T) {
	t.Parallel()
	st, _ := newStore(t)

	if err := st.Save(t.Context(), domain.Snapshot{Source: "--"}); !errors.Is(err, store.ErrSourceName) {
		t.Errorf("Save() error = %v, want store.ErrSourceName", err)
	}
	if _, err := st.Load(t.Context(), "--"); !errors.Is(err, store.ErrSourceName) {
		t.Errorf("Load() error = %v, want store.ErrSourceName", err)
	}
}

func TestNewRejectsAnIncompleteConfig(t *testing.T) {
	t.Parallel()
	valid := objectstore.Options{
		Bucket: bucket, AccessKey: accessKey, SecretKey: config.Secret(secretKey),
		HTTP: &http.Client{},
	}
	for _, tt := range []struct {
		name   string
		change func(*objectstore.Options)
	}{
		{"no bucket", func(o *objectstore.Options) { o.Bucket = "" }},
		{"no access key", func(o *objectstore.Options) { o.AccessKey = "" }},
		{"no secret key", func(o *objectstore.Options) { o.SecretKey = nil }},
		{"no HTTP client", func(o *objectstore.Options) { o.HTTP = nil }},
		{"plain HTTP endpoint", func(o *objectstore.Options) { o.Endpoint = "http://s3.fr-par.scw.cloud" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := valid
			tt.change(&opts)
			if _, err := objectstore.New(opts); !errors.Is(err, objectstore.ErrInvalidOptions) {
				t.Errorf("New() error = %v, want ErrInvalidOptions", err)
			}
		})
	}
	if _, err := objectstore.New(valid); err != nil {
		t.Errorf("New() with defaults error = %v", err)
	}
}
