// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"testing"
)

var fixtures = os.DirFS("../../../../testdata/rdap")

func readFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	b, err := fs.ReadFile(fixtures, name)
	if err != nil {
		tb.Fatalf("read fixture: %v", err)
	}
	return b
}

func TestParseDirectoryServer(t *testing.T) {
	t.Parallel()
	dir, err := parseDirectory(readFixture(t, "iana-dns.json"))
	if err != nil {
		t.Fatalf("parseDirectory() error = %v", err)
	}
	tests := []struct {
		name   string
		domain string
		want   string
		wantOK bool
	}{
		{"fr", "afnic.fr", "https://rdap.nic.fr/", true},
		{"com with path", "verisign.com", "https://rdap.verisign.com/com/v1/", true},
		{"second-level suffix", "exemple.co.re", "https://rdap.nic.re/", true},
		{"TLD missing from the file", "exemple.io", "", false},
		{"HTTP only server", "exemple.kg", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := dir.server(tt.domain)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("server(%q) = %q, %t, want %q, %t", tt.domain, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestParseDirectoryLongestMatch(t *testing.T) {
	t.Parallel()
	body := `{"version":"1.0","services":[
		[["uk"],["https://uk.example/"]],
		[["CO.UK"],["https://co-uk.example"]]]}`
	dir, err := parseDirectory([]byte(body))
	if err != nil {
		t.Fatalf("parseDirectory() error = %v", err)
	}
	if got, _ := dir.server("exemple.co.uk"); got != "https://co-uk.example/" {
		t.Errorf("server(exemple.co.uk) = %q, want the longest entry with a final slash", got)
	}
	if got, _ := dir.server("exemple.uk"); got != "https://uk.example/" {
		t.Errorf("server(exemple.uk) = %q, want https://uk.example/", got)
	}
}

func TestParseDirectoryHosts(t *testing.T) {
	t.Parallel()
	dir, err := parseDirectory(readFixture(t, "iana-dns.json"))
	if err != nil {
		t.Fatalf("parseDirectory() error = %v", err)
	}
	want := []string{"rdap.centralnic.com", "rdap.nic.fr", "rdap.nic.re", "rdap.verisign.com"}
	if got := dir.hosts(); !slices.Equal(got, want) {
		t.Errorf("hosts() = %v, want %v", got, want)
	}
}

func TestParseDirectoryErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{"not JSON", `<html>`},
		{"unknown version", `{"version":"2.0","services":[[["fr"],["https://rdap.nic.fr/"]]]}`},
		{"no HTTPS service", `{"version":"1.0","services":[[["kg"],["http://rdap.cctld.kg/"]]]}`},
		{"URL with credentials", `{"version":"1.0","services":[[["fr"],["https://u:p@rdap.nic.fr/"]]]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseDirectory([]byte(tt.body)); !errors.Is(err, ErrBootstrap) {
				t.Errorf("parseDirectory() error = %v, want ErrBootstrap", err)
			}
		})
	}
}
