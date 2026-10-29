// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"fmt"
	"net/url"
	"strings"
)

// allowlist is a set of exact hosts, each a lower-case name or IP address
// with an optional port, as in URL.Host.
type allowlist map[string]struct{}

func newAllowlist(hosts []string) (allowlist, error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("%w: no allowed host", ErrInvalidConfig)
	}
	list := make(allowlist, len(hosts))
	for _, h := range hosts {
		u, err := url.Parse("https://" + h)
		if h == "" || err != nil || u.Host != h || u.User != nil || u.Path != "" || u.RawQuery != "" {
			return nil, fmt.Errorf("%w: allowed host %q is not a bare host", ErrInvalidConfig, h)
		}
		list[strings.ToLower(h)] = struct{}{}
	}
	return list, nil
}

// allows reports whether u is an HTTPS URL without credentials on an
// allowed host. The port is part of the host: rdap.nic.fr does not allow
// rdap.nic.fr:443.
func (l allowlist) allows(u *url.URL) bool {
	if u.Scheme != "https" || u.User != nil {
		return false
	}
	_, ok := l[strings.ToLower(u.Host)]
	return ok
}
