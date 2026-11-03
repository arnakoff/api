// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

func TestRunRequiresTrustedProxiesWithLiveKeys(t *testing.T) {
	t.Parallel()
	if err := run(t.Context(), config.Config{APIKeyEnv: "live"}); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("run() without ARNAKOFF_TRUSTED_PROXIES error = %v, want ErrInvalid", err)
	}
}

func TestNewDemoHasAVerifierOnlyWhenOn(t *testing.T) {
	t.Parallel()
	limits := config.Demo{RateLimit: config.DefaultRateLimitDemo, RateLimitGlobal: config.DefaultRateLimitDemoGlobal}
	on := limits
	on.Origin, on.TurnstileSecret = "https://arnakoff.com", config.Secret("s3cr3t")
	for _, tt := range []struct {
		name string
		cfg  config.Demo
		want bool
	}{
		{"off", limits, false},
		{"on", on, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			demo, err := newDemo(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got := demo.Verifier != nil; got != tt.want {
				t.Errorf("verifier set = %t, want %t", got, tt.want)
			}
			if demo.Origin != tt.cfg.Origin || demo.Limit == nil || demo.GlobalLimit == nil {
				t.Errorf("demo = %+v, want the origin and both limits", demo)
			}
		})
	}
}

func TestProbeAsksTheConfiguredAddress(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	if err := probe(t.Context(), config.Config{Addr: srv.Listener.Addr().String()}); err != nil {
		t.Errorf("probe() error = %v, want nil", err)
	}
}
