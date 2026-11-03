// SPDX-License-Identifier: AGPL-3.0-only

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/arnakoff/api/internal/adapter/httpapi"
)

func TestClientKey(t *testing.T) {
	t.Parallel()

	proxies := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}
	tests := []struct {
		name    string
		remote  string
		forward []string // the X-Forwarded-For field lines
		trusted []netip.Prefix
		want    string
	}{
		{name: "direct client", remote: "203.0.113.7:51000", want: "203.0.113.7"},
		{name: "forwarded field ignored without trusted proxies", remote: "203.0.113.7:51000", forward: []string{"198.51.100.1"}, want: "203.0.113.7"},
		{name: "forwarded field ignored from an untrusted peer", remote: "203.0.113.7:51000", forward: []string{"198.51.100.1"}, trusted: proxies, want: "203.0.113.7"},
		{name: "client behind a trusted proxy", remote: "10.1.2.3:443", forward: []string{"198.51.100.1"}, trusted: proxies, want: "198.51.100.1"},
		{name: "rightmost untrusted address, not a spoofed one", remote: "10.1.2.3:443", forward: []string{"192.0.2.66, 198.51.100.1, 10.9.9.9"}, trusted: proxies, want: "198.51.100.1"},
		{name: "fields joined in order", remote: "10.1.2.3:443", forward: []string{"192.0.2.66", "198.51.100.1"}, trusted: proxies, want: "198.51.100.1"},
		{name: "every hop trusted", remote: "10.1.2.3:443", forward: []string{"10.4.4.4, 10.5.5.5"}, trusted: proxies, want: "10.4.4.4"},
		{name: "no forwarded field from a proxy", remote: "10.1.2.3:443", trusted: proxies, want: "10.1.2.3"},
		{name: "malformed hop stops the walk", remote: "10.1.2.3:443", forward: []string{"198.51.100.1, not-an-ip"}, trusted: proxies, want: "10.1.2.3"},
		{name: "IPv6 grouped by /64", remote: "[2001:db8:1:2:aaaa::1]:51000", want: "2001:db8:1:2::/64"},
		{name: "IPv6 behind a proxy", remote: "[fd00::1]:443", forward: []string{"2001:db8:1:2::99"}, trusted: proxies, want: "2001:db8:1:2::/64"},
		{name: "IPv4-mapped IPv6 as IPv4", remote: "[::ffff:203.0.113.7]:51000", want: "203.0.113.7"},
		{name: "remote address without port", remote: "unix", want: "invalid IP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/signals", nil)
			req.RemoteAddr = tt.remote
			for _, f := range tt.forward {
				req.Header.Add("X-Forwarded-For", f)
			}
			if got := httpapi.ClientKey(req, httpapi.ClientSource{Trusted: tt.trusted}); got != tt.want {
				t.Errorf("ClientKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientKeyFromAPlatformHeader(t *testing.T) {
	t.Parallel()

	src := httpapi.ClientSource{Header: "X-Real-Ip"}
	tests := []struct {
		name   string
		values []string // the X-Real-IP field lines
		want   string
	}{
		{name: "client the platform wrote", values: []string{"198.51.100.1"}, want: "198.51.100.1"},
		{name: "spaces around it", values: []string{" 198.51.100.1 "}, want: "198.51.100.1"},
		{name: "IPv6 grouped by /64", values: []string{"2001:db8:1:2:aaaa::1"}, want: "2001:db8:1:2::/64"},
		{name: "IPv4-mapped IPv6 as IPv4", values: []string{"::ffff:198.51.100.1"}, want: "198.51.100.1"},
		{name: "absent: the peer", want: "10.1.2.3"},
		{name: "empty: the peer", values: []string{""}, want: "10.1.2.3"},
		{name: "malformed: the peer", values: []string{"not-an-ip"}, want: "10.1.2.3"},
		{name: "a list: the peer", values: []string{"192.0.2.66, 198.51.100.1"}, want: "10.1.2.3"},
		{name: "two lines: the peer", values: []string{"192.0.2.66", "198.51.100.1"}, want: "10.1.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/signals", nil)
			req.RemoteAddr = "10.1.2.3:443"
			req.Header.Add("X-Forwarded-For", "192.0.2.99")
			for _, v := range tt.values {
				req.Header.Add("X-Real-IP", v)
			}
			if got := httpapi.ClientKey(req, src); got != tt.want {
				t.Errorf("ClientKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
