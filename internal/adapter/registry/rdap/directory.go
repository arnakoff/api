// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// bootstrapVersion is the only version of the bootstrap file format, RFC
// 9224 section 3.
const bootstrapVersion = "1.0"

// directory maps the domain names of the IANA bootstrap file to the base
// URL of their registry's RDAP server.
type directory map[string]string

// bootstrapFile holds the fields read from the IANA bootstrap file: each
// service is a list of entries (TLDs) and a list of base URLs.
type bootstrapFile struct {
	Version  string       `json:"version"`
	Services [][][]string `json:"services"`
}

// parseDirectory reads the IANA bootstrap file for domain names (RFC 9224).
// It keeps the first HTTPS base URL of each service, with a final slash,
// skips a service without one, and returns ErrBootstrap if none is left.
func parseDirectory(body []byte) (directory, error) {
	var f bootstrapFile
	if err := json.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBootstrap, err)
	}
	if f.Version != bootstrapVersion {
		return nil, fmt.Errorf("%w: version %q", ErrBootstrap, f.Version)
	}
	dir := make(directory)
	for _, s := range f.Services {
		if len(s) != 2 {
			continue
		}
		base, ok := firstHTTPS(s[1])
		if !ok {
			continue
		}
		for _, entry := range s[0] {
			dir[strings.ToLower(entry)] = base
		}
	}
	if len(dir) == 0 {
		return nil, fmt.Errorf("%w: no HTTPS service", ErrBootstrap)
	}
	return dir, nil
}

// firstHTTPS returns the first plain HTTPS URL of urls, with a final slash.
// A URL with credentials, a query or a fragment is not a base URL.
func firstHTTPS(urls []string) (string, bool) {
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			continue
		}
		if !strings.HasSuffix(raw, "/") {
			raw += "/"
		}
		return raw, true
	}
	return "", false
}

// server returns the base URL for name, a lower-case A-label domain name:
// the one of its longest suffix, made of whole labels, that the file lists.
func (d directory) server(name string) (string, bool) {
	for suffix := name; ; {
		if base, ok := d[suffix]; ok {
			return base, true
		}
		_, rest, found := strings.Cut(suffix, ".")
		if !found {
			return "", false
		}
		suffix = rest
	}
}

// hosts returns the hosts of every base URL, sorted, without duplicates:
// the allowlist of the client that queries the registries.
func (d directory) hosts() []string {
	var hosts []string
	for _, base := range d {
		u, err := url.Parse(base)
		if err != nil {
			continue // parseDirectory keeps parsed URLs only
		}
		hosts = append(hosts, u.Host)
	}
	slices.Sort(hosts)
	return slices.Compact(hosts)
}
