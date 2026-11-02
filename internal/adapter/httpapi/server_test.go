// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/platform/logging/loggingtest"
)

func TestNewServerBoundsEveryPhase(t *testing.T) {
	t.Parallel()

	logger, _ := loggingtest.New(t)
	srv := httpapi.NewServer(http.NotFoundHandler(), logger)
	if srv.ErrorLog == nil {
		t.Error("ErrorLog = nil, want the logger")
	}
	for name, d := range map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout, "ReadTimeout": srv.ReadTimeout,
		"WriteTimeout": srv.WriteTimeout, "IdleTimeout": srv.IdleTimeout,
	} {
		if d <= 0 {
			t.Errorf("%s = %v, want a bound", name, d)
		}
	}
	if srv.MaxHeaderBytes <= 0 || srv.MaxHeaderBytes >= http.DefaultMaxHeaderBytes {
		t.Errorf("MaxHeaderBytes = %d, want a bound under the default %d", srv.MaxHeaderBytes, http.DefaultMaxHeaderBytes)
	}
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func TestServeFinishesInFlightRequestsOnCancel(t *testing.T) {
	t.Parallel()

	started, release := make(chan struct{}), make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "done")
	})
	logger, _ := loggingtest.New(t)
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() { served <- httpapi.Serve(ctx, httpapi.NewServer(slow, logger), ln) }()

	got := make(chan string, 1)
	go func() {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+ln.Addr().String()+"/", nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			got <- err.Error()
			return
		}
		defer func() { _ = res.Body.Close() }()
		b, _ := io.ReadAll(res.Body)
		got <- string(b)
	}()
	<-started
	cancel()
	close(release)

	if body := <-got; body != "done" {
		t.Errorf("in-flight request got %q, want done", body)
	}
	if err := <-served; err != nil {
		t.Errorf("Serve = %v, want nil after a clean shutdown", err)
	}
}

func TestServeReturnsWhenTheListenerFails(t *testing.T) {
	t.Parallel()

	logger, _ := loggingtest.New(t)
	ln := listen(t)
	_ = ln.Close()
	if err := httpapi.Serve(t.Context(), httpapi.NewServer(http.NotFoundHandler(), logger), ln); err == nil {
		t.Error("Serve on a closed listener = nil, want an error")
	}
}
