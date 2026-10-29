// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"net/url"
	"testing"
)

func TestAllowlistAllows(t *testing.T) {
	t.Parallel()
	list, err := newAllowlist([]string{"RDAP.nic.fr", "127.0.0.1:8443"})
	if err != nil {
		t.Fatalf("newAllowlist() error = %v", err)
	}

	tests := []struct {
		url  string
		want bool
	}{
		{"https://rdap.nic.fr/domain/exemple.fr", true},
		{"https://Rdap.NIC.fr/domain/exemple.fr", true},
		{"https://127.0.0.1:8443/", true},
		{"https://rdap.nic.fr:443/", false},
		{"https://127.0.0.1/", false},
		{"https://nic.fr/", false},
		{"https://rdap.nic.fr.example.org/", false},
		{"https://rdap.nic.fr./", false},
		{"http://rdap.nic.fr/", false},
		{"HTTPS://rdap.nic.fr/", true},
		{"https://user@rdap.nic.fr/", false},
		{"//rdap.nic.fr/", false},
	}
	for _, tt := range tests {
		u, err := url.Parse(tt.url)
		if err != nil {
			t.Fatalf("url.Parse(%q) error = %v", tt.url, err)
		}
		if got := list.allows(u); got != tt.want {
			t.Errorf("allows(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}
