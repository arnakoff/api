// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

// smtpPassword stands for the password of the sending mailbox.
const smtpPassword = "m4ilb0x-p4ss"

// keyRequestVars turn the key requests on, with the demo they need.
func keyRequestVars() map[string]string {
	return map[string]string{
		"ARNAKOFF_LOG_KEY": validKey, "ARNAKOFF_DEMO_ORIGIN": "https://arnakoff.com",
		"ARNAKOFF_TURNSTILE_SECRET": turnstileSecret,
		"ARNAKOFF_SMTP_ADDR":        "smtp.mail.ovh.net:465", "ARNAKOFF_SMTP_USERNAME": "contact@arnakoff.com",
		"ARNAKOFF_SMTP_PASSWORD": smtpPassword,
	}
}

func TestLoadLeavesTheKeyRequestsOffByDefault(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{"ARNAKOFF_LOG_KEY": validKey}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	k := cfg.KeyRequest
	if k.Enabled() {
		t.Errorf("KeyRequest = %+v, want off without a server", k)
	}
	if k.MailFrom != config.DefaultMailFrom || k.LinkURL != config.DefaultLoginLinkURL ||
		k.RateLimit != config.DefaultRateLimitKeyRequest || k.RateLimitAddress != config.DefaultRateLimitKeyRequestAddress ||
		k.RateLimitGlobal != config.DefaultRateLimitKeyRequestGlobal {
		t.Errorf("KeyRequest = %+v, want the defaults", k)
	}
}

func TestLoadReadsTheKeyRequests(t *testing.T) {
	t.Parallel()
	vars := keyRequestVars()
	maps.Copy(vars, map[string]string{
		"ARNAKOFF_MAIL_FROM": "cles@arnakoff.com", "ARNAKOFF_LOGIN_LINK_URL": "https://preprod.arnakoff.com/cle",
		"ARNAKOFF_RATE_LIMIT_KEY_REQUEST": "7", "ARNAKOFF_RATE_LIMIT_KEY_REQUEST_ADDRESS": "2",
		"ARNAKOFF_RATE_LIMIT_KEY_REQUEST_GLOBAL": "45",
	})
	cfg, err := config.Load(env(vars))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	k := cfg.KeyRequest
	if !k.Enabled() || k.SMTPAddr != "smtp.mail.ovh.net:465" || k.SMTPUsername != "contact@arnakoff.com" ||
		string(k.SMTPPassword.Bytes()) != smtpPassword {
		t.Errorf("KeyRequest = %+v, want on with the server", k)
	}
	if k.MailFrom != "cles@arnakoff.com" || k.LinkURL != "https://preprod.arnakoff.com/cle" || k.RateLimit != 7 || k.RateLimitAddress != 2 ||
		k.RateLimitGlobal != 45 {
		t.Errorf("KeyRequest = %+v, want the values set", k)
	}
}

func TestLoadRejectsInvalidKeyRequests(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		set  map[string]string
		want string
	}{
		{"no port", map[string]string{"ARNAKOFF_SMTP_ADDR": "smtp.mail.ovh.net"}, "ARNAKOFF_SMTP_ADDR"},
		{"no host", map[string]string{"ARNAKOFF_SMTP_ADDR": ":465"}, "ARNAKOFF_SMTP_ADDR"},
		{"no username", map[string]string{"ARNAKOFF_SMTP_USERNAME": ""}, "ARNAKOFF_SMTP_USERNAME"},
		{"no password", map[string]string{"ARNAKOFF_SMTP_PASSWORD": ""}, "ARNAKOFF_SMTP_PASSWORD"},
		{"no demo", map[string]string{"ARNAKOFF_DEMO_ORIGIN": "", "ARNAKOFF_TURNSTILE_SECRET": ""}, "ARNAKOFF_TURNSTILE_SECRET"},
		{"http link", map[string]string{"ARNAKOFF_LOGIN_LINK_URL": "http://arnakoff.com/cle"}, "ARNAKOFF_LOGIN_LINK_URL"},
		{"link fragment", map[string]string{"ARNAKOFF_LOGIN_LINK_URL": "https://arnakoff.com/cle#x"}, "ARNAKOFF_LOGIN_LINK_URL"},
		{"link query", map[string]string{"ARNAKOFF_LOGIN_LINK_URL": "https://arnakoff.com/cle?a=1"}, "ARNAKOFF_LOGIN_LINK_URL"},
		{"limit", map[string]string{"ARNAKOFF_RATE_LIMIT_KEY_REQUEST_ADDRESS": "0"}, "ARNAKOFF_RATE_LIMIT_KEY_REQUEST_ADDRESS"},
		{"global limit", map[string]string{"ARNAKOFF_RATE_LIMIT_KEY_REQUEST_GLOBAL": "0"}, "ARNAKOFF_RATE_LIMIT_KEY_REQUEST_GLOBAL"},
	} {
		vars := keyRequestVars()
		maps.Copy(vars, tt.set)
		_, err := config.Load(env(vars))
		if !errors.Is(err, config.ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error = %v, want ErrInvalid naming %s", tt.name, err, tt.want)
		}
		if err != nil && strings.Contains(err.Error(), smtpPassword) {
			t.Errorf("%s: error %q quotes the password", tt.name, err)
		}
	}
}
