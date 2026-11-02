// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Environment variables of the demo access, read by Load.
const (
	EnvDemoOrigin          = "ARNAKOFF_DEMO_ORIGIN"
	EnvTurnstileSecret     = "ARNAKOFF_TURNSTILE_SECRET" //nolint:gosec // G101: the name of a variable, not its value.
	EnvRateLimitDemo       = "ARNAKOFF_RATE_LIMIT_DEMO"
	EnvRateLimitDemoGlobal = "ARNAKOFF_RATE_LIMIT_DEMO_GLOBAL"
)

// Defaults of the demo limits, in requests per minute, as the owner set
// them (#60).
const (
	// DefaultRateLimitDemo is the number of demo checks of a client
	// address.
	DefaultRateLimitDemo = 10

	// DefaultRateLimitDemoGlobal is the number of demo checks of every
	// client together: a flood from many addresses stays bounded.
	DefaultRateLimitDemoGlobal = 300
)

// Demo configures the demo access of the site's checker: checks without
// an API key, from the site only, each proven by a Cloudflare Turnstile
// token.
type Demo struct {
	// Origin is the site's origin, "https://host", the only one CORS
	// allows; empty when the demo is off.
	Origin string

	// TurnstileSecret is the secret key of the site's Turnstile widget;
	// nil when the demo is off.
	TurnstileSecret Secret

	// RateLimit is the number of demo checks per minute of a client
	// address, from 1 to MaxRateLimit.
	RateLimit int

	// RateLimitGlobal is the number of demo checks per minute of every
	// client together, from 1 to MaxRateLimit.
	RateLimitGlobal int
}

// Enabled reports whether the demo access is on.
func (d Demo) Enabled() bool {
	return d.TurnstileSecret != nil
}

// Hostname returns the host of Origin, without a port: the hostname a
// valid Turnstile token names.
func (d Demo) Hostname() string {
	u, err := url.Parse(d.Origin)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// parseDemo sets *d from the variables of the demo, read with lookup. The
// origin and the secret are set together or not at all.
func parseDemo(lookup func(string) (string, bool), d *Demo) error {
	err := errors.Join(
		parse(lookup, EnvDemoOrigin, "", &d.Origin, parseOrigin),
		parse(lookup, EnvTurnstileSecret, "", &d.TurnstileSecret, parseOptionalSecret),
		parse(lookup, EnvRateLimitDemo, strconv.Itoa(DefaultRateLimitDemo), &d.RateLimit, parseRateLimit),
		parse(lookup, EnvRateLimitDemoGlobal, strconv.Itoa(DefaultRateLimitDemoGlobal), &d.RateLimitGlobal, parseRateLimit),
	)
	if err != nil {
		return err
	}
	if (d.Origin == "") != (d.TurnstileSecret == nil) {
		return fmt.Errorf("%w: %s and %s: set both to turn the demo on, or neither",
			ErrInvalid, EnvDemoOrigin, EnvTurnstileSecret)
	}
	return nil
}

// parseOrigin accepts an empty string, as no origin, or an HTTPS origin as
// a browser sends it in the Origin field: "https://host" or
// "https://host:port", in lower case, without a path.
func parseOrigin(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	u, err := parseHTTPS(s)
	switch {
	case err != nil:
		return "", err
	case s != "https://"+u.Host:
		return "", fmt.Errorf("%q is not an origin: https:// and a host only", s)
	case s != strings.ToLower(s):
		return "", fmt.Errorf("%q is not in lower case", s)
	}
	return s, nil
}
