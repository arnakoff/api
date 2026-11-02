// SPDX-License-Identifier: AGPL-3.0-only

package hosting_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/reference/hosting"
	"github.com/arnakoff/api/internal/domain"
)

func standsForSite(t *testing.T, p domain.Platforms, host string) bool {
	t.Helper()
	d, err := domain.ParseDomain(host)
	if err != nil {
		t.Fatalf("ParseDomain(%q): %v", host, err)
	}
	return p.PageStandsForSite(d)
}

func TestParseReadsEveryPlatform(t *testing.T) {
	t.Parallel()

	const csv = "domain,proof\n" +
		"linktr.ee,https://linktr.ee/linktree\n" +
		"facebook.com,https://www.facebook.com/facebook\n"
	p, err := hosting.Parse(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for host, want := range map[string]bool{"linktr.ee": false, "www.facebook.com": false, "coinhouse.com": true} {
		if got := standsForSite(t, p, host); got != want {
			t.Errorf("PageStandsForSite(%s) = %v; want %v", host, got, want)
		}
	}
}

func TestParseRejectsMalformedFiles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		csv  string
		want error
	}{
		"other header":    {"domain,of,proof\nt.me,x,https://t.me/telegram\n", hosting.ErrFormat},
		"missing column":  {"domain,proof\nt.me\n", hosting.ErrFormat},
		"empty":           {"", hosting.ErrFormat},
		"proof not HTTPS": {"domain,proof\nt.me,http://t.me/telegram\n", hosting.ErrFormat},
		"proof elsewhere": {"domain,proof\nt.me,https://telegram.org/\n", hosting.ErrFormat},
		"not registrable": {"domain,proof\nwww.t.me,https://www.t.me/telegram\n", domain.ErrInvalidDomain},
		"listed twice":    {"domain,proof\nt.me,https://t.me/a\nt.me,https://t.me/b\n", domain.ErrInvalidDomain},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := hosting.Parse(strings.NewReader(tt.csv)); !errors.Is(err, tt.want) {
				t.Errorf("Parse() error = %v; want %v", err, tt.want)
			}
		})
	}
}

// TestLoadReadsTheShippedList checks that the embedded list is valid and
// leaves the providers' own sites alone.
func TestLoadReadsTheShippedList(t *testing.T) {
	t.Parallel()

	p, err := hosting.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for host, want := range map[string]bool{"linktr.ee": false, "t.me": false, "www.coinhouse.com": true} {
		if got := standsForSite(t, p, host); got != want {
			t.Errorf("PageStandsForSite(%s) = %v; want %v", host, got, want)
		}
	}
}
