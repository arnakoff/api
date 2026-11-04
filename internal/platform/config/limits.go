// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
	"net/netip"
	"net/textproto"
	"strings"
)

// Defaults of the rate limits, in requests per minute, as the owner set
// them until the offers of #80.
const (
	DefaultRateLimitKey = 60
	DefaultRateLimitIP  = 30

	// DefaultRateLimitAuth is the number of refused API keys a minute of
	// a client address, as the owner set it (#197).
	DefaultRateLimitAuth = 30

	// MaxRateLimit is the highest limit Load accepts.
	MaxRateLimit = 10000
)

// noProxy is the value of ARNAKOFF_TRUSTED_PROXIES of an API that no
// proxy stands in front of.
const noProxy = "none"

// RequireTrustedProxies returns an error, wrapping ErrInvalid, when the
// API serves live keys without ARNAKOFF_TRUSTED_PROXIES nor
// ARNAKOFF_CLIENT_IP_HEADER: the operator must say who tells the client's
// address, or that no proxy does.
func (c Config) RequireTrustedProxies() error {
	if c.APIKeyEnv == "live" && c.TrustedProxies == nil && c.ClientIPHeader == "" {
		return fmt.Errorf("%w: %s or %s: one is required with live keys, a list of CIDR prefixes or %q, or a field name",
			ErrInvalid, EnvTrustedProxies, EnvClientIPHeader, noProxy)
	}
	return nil
}

// parseClientIPHeader sets *field from ARNAKOFF_CLIENT_IP_HEADER, read
// with lookup: a field name, kept in canonical form. It refuses one set
// with ARNAKOFF_TRUSTED_PROXIES, read into *trusted before: the two tell
// the client's address in different ways.
func parseClientIPHeader(lookup func(string) (string, bool), field *string, trusted *[]netip.Prefix) error {
	if err := parse(lookup, EnvClientIPHeader, "", field, parseFieldName); err != nil {
		return err
	}
	if *field != "" && *trusted != nil {
		return fmt.Errorf("%w: %s: not with %s, set one of them", ErrInvalid, EnvClientIPHeader, EnvTrustedProxies)
	}
	return nil
}

// parseFieldName accepts an empty string, as none, or an HTTP field name
// of letters, digits and hyphens, and returns it in canonical form.
func parseFieldName(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if strings.IndexFunc(s, isNotFieldChar) >= 0 {
		return "", fmt.Errorf("%q is not a field name of letters, digits and hyphens", s)
	}
	return textproto.CanonicalMIMEHeaderKey(s), nil
}

// isNotFieldChar reports whether c is neither an ASCII letter, a digit nor
// a hyphen.
func isNotFieldChar(c rune) bool {
	return c != '-' && (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z')
}

// parseRateLimit accepts a decimal number from 1 to MaxRateLimit.
var parseRateLimit = intFrom1To(MaxRateLimit)

// parseTrustedProxies sets *field from ARNAKOFF_TRUSTED_PROXIES, read
// with lookup, and leaves it nil when the variable is unset: it has no
// default.
func parseTrustedProxies(lookup func(string) (string, bool), field *[]netip.Prefix) error {
	if _, ok := lookup(EnvTrustedProxies); !ok {
		return nil
	}
	return parse(lookup, EnvTrustedProxies, "", field, parsePrefixes)
}

// parsePrefixes accepts noProxy, as an empty list, or a comma-separated
// list of CIDR prefixes without host bits.
func parsePrefixes(s string) ([]netip.Prefix, error) {
	switch s {
	case "":
		return nil, errEmptyProxies
	case noProxy:
		return []netip.Prefix{}, nil
	}
	var prefixes []netip.Prefix
	for field := range strings.SplitSeq(s, ",") {
		p, err := netip.ParsePrefix(strings.TrimSpace(field))
		if err != nil {
			return nil, fmt.Errorf("%q is not a CIDR prefix", field)
		}
		if p != p.Masked() {
			return nil, fmt.Errorf("%q has host bits set, want %s", field, p.Masked())
		}
		prefixes = append(prefixes, p)
	}
	return prefixes, nil
}

// errEmptyProxies refuses a set but empty ARNAKOFF_TRUSTED_PROXIES.
var errEmptyProxies = errors.New(`empty, want a list of CIDR prefixes or "none"`)
