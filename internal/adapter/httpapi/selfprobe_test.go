// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
	"github.com/arnakoff/api/internal/platform/httpclient/httpclienttest"
)

// listening starts h on a loopback port and returns that port.
func listening(t *testing.T, h http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func TestProbeAliveReachesTheServedHealthz(t *testing.T) {
	t.Parallel()
	opts, _ := options(t, false)
	port := listening(t, httpapi.NewHandler(opts))

	// The probe runs next to the server: an address listening on every
	// interface is probed on loopback.
	for _, addr := range []string{":" + port, "0.0.0.0:" + port, "[::]:" + port, "127.0.0.1:" + port} {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()
			if err := httpapi.ProbeAlive(t.Context(), addr); err != nil {
				t.Errorf("ProbeAlive(%q) error = %v, want nil", addr, err)
			}
		})
	}
}

func TestProbeAliveFails(t *testing.T) {
	t.Parallel()
	unavailable := listening(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	redirected := listening(t, http.RedirectHandler("/elsewhere", http.StatusFound))
	stopped := httptest.NewServer(http.NotFoundHandler())
	stopped.Close()
	for _, tt := range []struct {
		name string
		addr string
	}{
		{"not 200", "127.0.0.1:" + unavailable},
		{"redirect not followed", "127.0.0.1:" + redirected},
		{"nothing listening", stopped.Listener.Addr().String()},
		{"invalid address", "localhost"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := httpapi.ProbeAlive(t.Context(), tt.addr); err == nil {
				t.Errorf("ProbeAlive(%q) = nil, want an error", tt.addr)
			}
		})
	}
}

func TestProbeAliveStopsWithItsContext(t *testing.T) {
	t.Parallel()
	port := listening(t, http.HandlerFunc(httpclienttest.Stall))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := httpapi.ProbeAlive(ctx, "127.0.0.1:"+port); err == nil {
		t.Error("ProbeAlive() with a cancelled context = nil, want an error")
	}
}
