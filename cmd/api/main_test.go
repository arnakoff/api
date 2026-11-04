// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
	"github.com/arnakoff/api/internal/domain"
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

func TestNewKeyRequestsHasARequesterOnlyWhenOn(t *testing.T) {
	t.Parallel()
	off := config.KeyRequest{
		MailFrom: config.DefaultMailFrom, LinkURL: config.DefaultLoginLinkURL,
		RateLimit: config.DefaultRateLimitKeyRequest, RateLimitAddress: config.DefaultRateLimitKeyRequestAddress,
	}
	on := off
	on.SMTPAddr, on.SMTPUsername, on.SMTPPassword = "smtp.mail.ovh.net:465", "contact@arnakoff.com", config.Secret("pw")
	for _, tt := range []struct {
		name string
		cfg  config.KeyRequest
		want bool
	}{
		{"off", off, false},
		{"on", on, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts, err := newKeyRequests(config.Config{KeyRequest: tt.cfg}, postgres.KeyServices{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := opts.Requests != nil; got != tt.want {
				t.Errorf("requester set = %t, want %t", got, tt.want)
			}
			if opts.Limit == nil || opts.AddressLimit == nil {
				t.Errorf("options = %+v, want both limits", opts)
			}
		})
	}
}

func TestNewKeyRequestsRefusesAnInvalidSender(t *testing.T) {
	t.Parallel()
	cfg := config.KeyRequest{
		SMTPAddr: "smtp.mail.ovh.net:465", SMTPUsername: "u", SMTPPassword: config.Secret("pw"),
		MailFrom: "contact.arnakoff.com", RateLimit: 1, RateLimitAddress: 1,
	}
	_, err := newKeyRequests(config.Config{KeyRequest: cfg}, postgres.KeyServices{}, nil)
	if !errors.Is(err, domain.ErrInvalidEmail) || !strings.Contains(err.Error(), config.EnvMailFrom) {
		t.Errorf("error = %v, want an invalid address naming %s", err, config.EnvMailFrom)
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
