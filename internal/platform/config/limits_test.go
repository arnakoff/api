// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

func TestLoadAppliesTheLimitDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RateLimitKey != config.DefaultRateLimitKey || cfg.RateLimitIP != config.DefaultRateLimitIP {
		t.Errorf("RateLimitKey, RateLimitIP = %d, %d, want %d, %d",
			cfg.RateLimitKey, cfg.RateLimitIP, config.DefaultRateLimitKey, config.DefaultRateLimitIP)
	}
	if cfg.RateLimitAuth != config.DefaultRateLimitAuth {
		t.Errorf("RateLimitAuth = %d, want %d", cfg.RateLimitAuth, config.DefaultRateLimitAuth)
	}
	if cfg.TrustedProxies != nil {
		t.Errorf("TrustedProxies = %v, want nil when unset", cfg.TrustedProxies)
	}
}

func TestLoadReadsTheLimits(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_RATE_LIMIT_KEY": "600", "ARNAKOFF_RATE_LIMIT_IP": "1",
		"ARNAKOFF_RATE_LIMIT_AUTH": "5",
		"ARNAKOFF_TRUSTED_PROXIES": "10.0.0.0/8, 2001:db8::/32",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RateLimitKey != 600 || cfg.RateLimitIP != 1 {
		t.Errorf("RateLimitKey, RateLimitIP = %d, %d, want 600, 1", cfg.RateLimitKey, cfg.RateLimitIP)
	}
	if cfg.RateLimitAuth != 5 {
		t.Errorf("RateLimitAuth = %d, want 5", cfg.RateLimitAuth)
	}
	want := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("2001:db8::/32")}
	if !slices.Equal(cfg.TrustedProxies, want) {
		t.Errorf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
}

func TestLoadReadsNoTrustedProxy(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_TRUSTED_PROXIES": "none"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TrustedProxies == nil || len(cfg.TrustedProxies) != 0 {
		t.Errorf("TrustedProxies = %#v, want empty and set", cfg.TrustedProxies)
	}
}

func TestLoadRejectsInvalidLimits(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, variable, value string }{
		{"key limit zero", "ARNAKOFF_RATE_LIMIT_KEY", "0"},
		{"key limit too high", "ARNAKOFF_RATE_LIMIT_KEY", "10001"},
		{"ip limit not a number", "ARNAKOFF_RATE_LIMIT_IP", "30/min"},
		{"auth limit zero", "ARNAKOFF_RATE_LIMIT_AUTH", "0"},
		{"empty proxies", "ARNAKOFF_TRUSTED_PROXIES", ""},
		{"proxy without length", "ARNAKOFF_TRUSTED_PROXIES", "10.0.0.1"},
		{"proxy with host bits", "ARNAKOFF_TRUSTED_PROXIES", "10.0.0.1/8"},
		{"empty proxy in the list", "ARNAKOFF_TRUSTED_PROXIES", "10.0.0.0/8,"},
		{"none among proxies", "ARNAKOFF_TRUSTED_PROXIES", "none,10.0.0.0/8"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey, tt.variable: tt.value}))
			if !errors.Is(err, config.ErrInvalid) {
				t.Errorf("Load(%s=%q) error = %v, want ErrInvalid", tt.variable, tt.value, err)
			}
		})
	}
}

func TestRequireTrustedProxies(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{"live without proxies", config.Config{APIKeyEnv: "live"}, true},
		{"live with none", config.Config{APIKeyEnv: "live", TrustedProxies: []netip.Prefix{}}, false},
		{"live with a client header", config.Config{APIKeyEnv: "live", ClientIPHeader: "X-Real-Ip"}, false},
		{"test without proxies", config.Config{APIKeyEnv: "test"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.RequireTrustedProxies()
			if got := errors.Is(err, config.ErrInvalid); got != tt.wantErr {
				t.Errorf("RequireTrustedProxies() = %v, want an ErrInvalid: %t", err, tt.wantErr)
			}
		})
	}
}

func TestLoadReadsTheClientIPHeader(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_CLIENT_IP_HEADER": "x-real-ip"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientIPHeader != "X-Real-Ip" {
		t.Errorf("ClientIPHeader = %q, want X-Real-Ip in canonical form", cfg.ClientIPHeader)
	}
	cfg, err = config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey}))
	if err != nil || cfg.ClientIPHeader != "" {
		t.Errorf("ClientIPHeader = %q, %v; want empty by default", cfg.ClientIPHeader, err)
	}
}

func TestLoadRejectsAnInvalidClientIPHeader(t *testing.T) {
	t.Parallel()
	for name, vars := range map[string]map[string]string{
		"not a field name": {"ARNAKOFF_CLIENT_IP_HEADER": "X Real IP"},
		"with proxies":     {"ARNAKOFF_CLIENT_IP_HEADER": "X-Real-IP", "ARNAKOFF_TRUSTED_PROXIES": "10.0.0.0/8"},
		"with none":        {"ARNAKOFF_CLIENT_IP_HEADER": "X-Real-IP", "ARNAKOFF_TRUSTED_PROXIES": "none"},
	} {
		vars["ARNAKOFF_LOG_KEY"] = validKey
		if _, err := config.Load(env(vars)); !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), "ARNAKOFF_CLIENT_IP_HEADER") {
			t.Errorf("%s: error = %v, want ErrInvalid naming ARNAKOFF_CLIENT_IP_HEADER", name, err)
		}
	}
}

func TestLoadRejectsABatchLargerThanTheKeyQuota(t *testing.T) {
	t.Parallel()
	_, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_RATE_LIMIT_KEY": "40", "ARNAKOFF_BATCH_MAX_ITEMS": "41",
	}))
	if !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), "ARNAKOFF_BATCH_MAX_ITEMS") {
		t.Errorf("Load(batch 41, key quota 40) error = %v, want ErrInvalid naming ARNAKOFF_BATCH_MAX_ITEMS", err)
	}
	if _, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_RATE_LIMIT_KEY": "40", "ARNAKOFF_BATCH_MAX_ITEMS": "40",
	})); err != nil {
		t.Errorf("Load(batch 40, key quota 40) error = %v, want nil", err)
	}
}
