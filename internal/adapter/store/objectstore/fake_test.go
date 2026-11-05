// SPDX-License-Identifier: AGPL-3.0-only

package objectstore_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// fakeS3 is a bucket server answering the path-style GetObject and
// PutObject requests of the adapter, with ETags and If-None-Match, enough
// to test it without the network.
type fakeS3 struct {
	t *testing.T

	mu        sync.Mutex
	objects   map[string][]byte // By path, "/<bucket>/<key>".
	downloads int               // GetObject answered with a body.
	notMod    int               // GetObject answered 304.
	lastAuth  string            // Authorization field of the last request.
	lastINM   string            // If-None-Match field of the last request.
	failWith  int               // When not 0, the status of every answer.
}

func newFakeS3(t *testing.T) *fakeS3 {
	t.Helper()
	return &fakeS3{t: t, objects: map[string][]byte{}}
}

// ServeHTTP implements http.Handler.
func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastAuth = r.Header.Get("Authorization")
	f.lastINM = r.Header.Get("If-None-Match")
	if f.failWith != 0 {
		w.WriteHeader(f.failWith)
		return
	}
	switch r.Method {
	case http.MethodPut:
		b, err := io.ReadAll(r.Body)
		if err != nil {
			f.t.Errorf("read PUT body: %v", err)
		}
		f.objects[r.URL.Path] = b
		w.Header().Set("ETag", etag(b))
	case http.MethodGet:
		f.get(w, r)
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (f *fakeS3) get(w http.ResponseWriter, r *http.Request) {
	b, ok := f.objects[r.URL.Path]
	if !ok {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`)
		return
	}
	tag := etag(b)
	w.Header().Set("ETag", tag)
	if r.Header.Get("If-None-Match") == tag {
		f.notMod++
		w.WriteHeader(http.StatusNotModified)
		return
	}
	f.downloads++
	_, _ = w.Write(b)
}

// put stores b at path, as another process would.
func (f *fakeS3) put(path string, b []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[path] = b
}

func (f *fakeS3) object(path string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.objects[path]
	return b, ok
}

func (f *fakeS3) counts() (downloads, notModified int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.downloads, f.notMod
}

func (f *fakeS3) auth() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastAuth
}

func (f *fakeS3) lastIfNoneMatch() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastINM
}

func (f *fakeS3) fail(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failWith = status
}

// etag is a quoted entity tag of b.
func etag(b []byte) string {
	sum := sha256.Sum256(b)
	return `"` + strings.ToLower(hex.EncodeToString(sum[:16])) + `"`
}
