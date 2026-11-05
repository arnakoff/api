// SPDX-License-Identifier: AGPL-3.0-only

package scaleway_test

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/arnakoff/api/internal/adapter/scaleway"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// fixtures are answers of the Scaleway API recorded by the Terraform
// provider (github.com/scaleway/terraform-provider-scaleway/v2, cassettes of
// internal/services/*/testdata): IAM and Secret Manager from v2.91.0,
// containers/v1beta1 from v2.60.0, the last version that called it. Their
// secret keys were already replaced by the recorder.
var fixtures = os.DirFS("../../../testdata/scaleway")

// call is a request the fake API received.
type call struct {
	method, path, query, body, auth string
}

// answer is what the fake API sends to a route.
type answer struct {
	status  int
	fixture string // empty for no body
}

// fakeAPI is a Scaleway API that answers each "METHOD /path" with the next
// of its answers, the last one repeated, and records every call.
type fakeAPI struct {
	t      *testing.T
	srv    *httptest.Server
	mu     sync.Mutex
	routes map[string][]answer
	calls  []call
}

func newFakeAPI(t *testing.T, routes map[string][]answer) *fakeAPI {
	t.Helper()
	f := &fakeAPI{t: t, routes: routes}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeAPI) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	route := r.Method + " " + r.URL.Path
	f.mu.Lock()
	f.calls = append(f.calls, call{r.Method, r.URL.Path, r.URL.RawQuery, string(body), r.Header.Get("X-Auth-Token")})
	answers := f.routes[route]
	if len(answers) > 1 {
		f.routes[route] = answers[1:]
	}
	f.mu.Unlock()
	if len(answers) == 0 {
		http.Error(w, `{"message":"no route `+route+`"}`, http.StatusNotFound)
		return
	}
	a := answers[0]
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(a.status)
	if a.fixture != "" {
		b, err := fs.ReadFile(fixtures, a.fixture)
		if err != nil {
			f.t.Error(err)
		}
		_, _ = w.Write(b)
	}
}

// recorded returns the calls received so far.
func (f *fakeAPI) recorded() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]call(nil), f.calls...)
}

// client returns a Scaleway client of the fake API, through the platform
// client.
func (f *fakeAPI) client() *scw.Client {
	f.t.Helper()
	c, err := scaleway.NewClient(scaleway.Options{
		HTTP:      httpclienttest.NewClient(f.t, f.srv, httpclient.Config{}),
		APIURL:    f.srv.URL,
		AccessKey: accessKey,
		SecretKey: config.Secret(secretKey),
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return c
}

const (
	accessKey = "SCWROTATORXXXXXXXXXX"
	secretKey = "22222222-2222-2222-2222-222222222222"
)
