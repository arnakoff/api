// SPDX-License-Identifier: AGPL-3.0-only

package httpclient_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/httpclient"
)

// token stands for the secret a webhook URL carries in its path.
const token = "T0K3N-s3cr3t"

func TestPostSendsTheBody(t *testing.T) {
	t.Parallel()
	type received struct{ method, contentType, userAgent, body string }
	got := make(chan received, 1)
	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- received{r.Method, r.Header.Get("Content-Type"), r.Header.Get("User-Agent"), string(b)}
		w.WriteHeader(http.StatusNoContent)
	})
	c := newClient(t, srv, httpclient.Config{UserAgent: "test-agent"})

	if err := c.Post(t.Context(), srv.URL+"/hook/"+token, "application/json", []byte(`{"a":1}`)); err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	want := received{http.MethodPost, "application/json", "test-agent", `{"a":1}`}
	if r := <-got; r != want {
		t.Errorf("server received %+v, want %+v", r, want)
	}
}

func TestPostRetriesAndSendsTheBodyAgain(t *testing.T) {
	t.Parallel()
	bodies := make(chan string, 2)
	srv, hits := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies <- string(b)
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	c := newClient(t, srv, httpclient.Config{Attempts: 2})

	if err := c.Post(t.Context(), srv.URL+"/hook", "text/plain", []byte("alerte")); err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if hits.Load() != 2 || <-bodies != "alerte" || <-bodies != "alerte" {
		t.Errorf("server got %d requests, want 2 with the same body", hits.Load())
	}
}

func TestPostErrorsNeverQuoteTheURL(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	c := newClient(t, srv, httpclient.Config{})
	closed, _ := newServer(t, func(http.ResponseWriter, *http.Request) {})
	closed.Close()
	cc := newClient(t, closed, httpclient.Config{})

	tests := []struct {
		name string
		c    *httpclient.Client
		url  string
		want func(error) bool
	}{
		{"status", c, srv.URL + "/hook/" + token, func(err error) bool {
			var se *httpclient.StatusError
			return errors.As(err, &se) && se.StatusCode == http.StatusForbidden
		}},
		{"not allowed", c, "https://other.example/hook/" + token, func(err error) bool {
			return errors.Is(err, httpclient.ErrURLNotAllowed)
		}},
		{"transport", cc, closed.URL + "/hook/" + token, func(err error) bool { return err != nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.c.Post(t.Context(), tt.url, "text/plain", nil)
			if !tt.want(err) {
				t.Fatalf("Post() error = %v, not the one expected", err)
			}
			if strings.Contains(err.Error(), token) {
				t.Errorf("Post() error %q quotes the URL", err)
			}
		})
	}
}
