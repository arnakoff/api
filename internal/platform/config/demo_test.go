// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

// turnstileSecret stands for the secret key of a Turnstile widget.
const turnstileSecret = "0x4AAAAAAA-s3cr3t" //nolint:gosec // G101: a fake secret of the tests.

func TestLoadLeavesTheDemoOffByDefault(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d := cfg.Demo
	if d.Enabled() || d.Origin != "" {
		t.Errorf("Demo = %+v, want off without an origin", d)
	}
	if d.RateLimit != config.DefaultRateLimitDemo || d.RateLimitGlobal != config.DefaultRateLimitDemoGlobal {
		t.Errorf("RateLimit, RateLimitGlobal = %d, %d, want %d, %d",
			d.RateLimit, d.RateLimitGlobal, config.DefaultRateLimitDemo, config.DefaultRateLimitDemoGlobal)
	}
}

func TestLoadReadsTheDemo(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_DEMO_ORIGIN": "https://arnakoff.com",
		"ARNAKOFF_TURNSTILE_SECRET": turnstileSecret,
		"ARNAKOFF_RATE_LIMIT_DEMO":  "5", "ARNAKOFF_RATE_LIMIT_DEMO_GLOBAL": "100",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d := cfg.Demo
	if !d.Enabled() || d.Origin != "https://arnakoff.com" || string(d.TurnstileSecret.Bytes()) != turnstileSecret {
		t.Errorf("Demo = %+v, want on for https://arnakoff.com with the secret", d)
	}
	if d.Hostname() != "arnakoff.com" {
		t.Errorf("Hostname() = %q, want arnakoff.com", d.Hostname())
	}
	if d.RateLimit != 5 || d.RateLimitGlobal != 100 {
		t.Errorf("RateLimit, RateLimitGlobal = %d, %d, want 5, 100", d.RateLimit, d.RateLimitGlobal)
	}
}

func TestLoadRejectsAnInvalidDemo(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		vars map[string]string
	}{
		{"origin without secret", map[string]string{"ARNAKOFF_DEMO_ORIGIN": "https://arnakoff.com"}},
		{"secret without origin", map[string]string{"ARNAKOFF_TURNSTILE_SECRET": turnstileSecret}},
		{"http origin", demoVars("http://arnakoff.com")},
		{"origin with a path", demoVars("https://arnakoff.com/")},
		{"origin with a query", demoVars("https://arnakoff.com?a=1")},
		{"origin with credentials", demoVars("https://user@arnakoff.com")},
		{"origin in upper case", demoVars("https://Arnakoff.com")},
		{"several origins", demoVars("https://arnakoff.com,https://www.arnakoff.com")},
		{"demo limit zero", map[string]string{"ARNAKOFF_RATE_LIMIT_DEMO": "0"}},
		{"global limit too high", map[string]string{"ARNAKOFF_RATE_LIMIT_DEMO_GLOBAL": "10001"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.vars["ARNAKOFF_LOG_KEY"] = validKey
			_, err := config.Load(env(tt.vars))
			if !errors.Is(err, config.ErrInvalid) {
				t.Fatalf("Load(%v) error = %v, want ErrInvalid", tt.vars, err)
			}
			if strings.Contains(err.Error(), turnstileSecret) {
				t.Errorf("Load error %q quotes the Turnstile secret", err)
			}
		})
	}
}

// demoVars returns the variables of a demo with the secret and origin.
func demoVars(origin string) map[string]string {
	return map[string]string{"ARNAKOFF_DEMO_ORIGIN": origin, "ARNAKOFF_TURNSTILE_SECRET": turnstileSecret}
}
