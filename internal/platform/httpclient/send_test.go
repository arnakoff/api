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

func TestSendSendsTheMethodHeadersAndBody(t *testing.T) {
	t.Parallel()
	type received struct{ method, auth, accept, userAgent, body string }
	got := make(chan received, 1)
	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- received{r.Method, r.Header.Get("Authorization"), r.Header.Get("Accept"), r.Header.Get("User-Agent"), string(b)}
		_, _ = io.WriteString(w, "answer")
	})
	c := newClient(t, srv, httpclient.Config{UserAgent: "test-agent"})
	header := http.Header{"Authorization": {"Bearer " + token}, "Accept": {"application/json"}}

	answer, err := c.Send(t.Context(), http.MethodPut, srv.URL+"/secret", header, []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	want := received{http.MethodPut, "Bearer " + token, "application/json", "test-agent", `{"a":1}`}
	if r := <-got; r != want || string(answer) != "answer" {
		t.Errorf("server received %+v, answered %q; want %+v", r, answer, want)
	}
}

func TestSendRetriesWithTheSameHeaders(t *testing.T) {
	t.Parallel()
	auths := make(chan string, 2)
	srv, hits := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		auths <- r.Header.Get("Authorization")
		if len(auths) == 1 {
			w.WriteHeader(http.StatusBadGateway)
		}
	})
	c := newClient(t, srv, httpclient.Config{Attempts: 2})

	if _, err := c.Send(t.Context(), http.MethodGet, srv.URL+"/key", http.Header{"Authorization": {"Bearer x"}}, nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if hits.Load() != 2 || <-auths != "Bearer x" || <-auths != "Bearer x" {
		t.Errorf("server got %d requests, want 2 with the header", hits.Load())
	}
}

func TestSendErrorsAreStatusErrorsWithoutTheURL(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	c := newClient(t, srv, httpclient.Config{})

	_, err := c.Send(t.Context(), http.MethodPut, srv.URL+"/secret/"+token, nil, nil)

	var se *httpclient.StatusError
	if !errors.As(err, &se) || se.StatusCode != http.StatusForbidden {
		t.Errorf("Send() error = %v, want a *StatusError 403", err)
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("error %q quotes the URL", err)
	}
}
