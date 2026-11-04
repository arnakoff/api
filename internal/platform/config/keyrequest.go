// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Environment variables of the key requests by login link, read by Load.
const (
	EnvSMTPAddr                   = "ARNAKOFF_SMTP_ADDR"
	EnvSMTPUsername               = "ARNAKOFF_SMTP_USERNAME"
	EnvSMTPPassword               = "ARNAKOFF_SMTP_PASSWORD" //nolint:gosec // G101: the name of a variable, not its value.
	EnvMailFrom                   = "ARNAKOFF_MAIL_FROM"
	EnvLoginLinkURL               = "ARNAKOFF_LOGIN_LINK_URL"
	EnvRateLimitKeyRequest        = "ARNAKOFF_RATE_LIMIT_KEY_REQUEST"
	EnvRateLimitKeyRequestAddress = "ARNAKOFF_RATE_LIMIT_KEY_REQUEST_ADDRESS"
)

// Defaults of the key requests.
const (
	// DefaultMailFrom is the sending mailbox the owner chose
	// (arnakoff/.github#3).
	DefaultMailFrom = "contact@arnakoff.com"

	// DefaultLoginLinkURL is the page of the site a login link opens.
	DefaultLoginLinkURL = "https://arnakoff.com/cle"

	// DefaultRateLimitKeyRequest is the number of key requests and
	// redemptions per minute of a client address.
	DefaultRateLimitKeyRequest = 5

	// DefaultRateLimitKeyRequestAddress is the number of login links an
	// email address may receive per hour.
	DefaultRateLimitKeyRequestAddress = 3
)

// KeyRequest configures the free API keys requested by login link (#59).
type KeyRequest struct {
	// SMTPAddr is the "host:port" of the sending server, which speaks TLS
	// at once; empty when the key requests are off.
	SMTPAddr string

	// SMTPUsername and SMTPPassword authenticate with the server.
	SMTPUsername string
	SMTPPassword Secret

	// MailFrom is the sender address, a bare address.
	MailFrom string

	// LinkURL is the HTTPS page of the site a login link opens, the token
	// in its fragment.
	LinkURL string

	// RateLimit is the number of requests per minute of a client address
	// to the key request routes, from 1 to MaxRateLimit.
	RateLimit int

	// RateLimitAddress is the number of login links per hour an email
	// address may receive, from 1 to MaxRateLimit.
	RateLimitAddress int
}

// Enabled reports whether the key requests are on.
func (k KeyRequest) Enabled() bool {
	return k.SMTPAddr != ""
}

// parseKeyRequest sets *k from its variables, read with lookup. A server
// needs its credentials, and the demo's Turnstile widget d, which proves
// each request; d is read once parsed.
func parseKeyRequest(lookup func(string) (string, bool), k *KeyRequest, d *Demo) error {
	err := errors.Join(
		parse(lookup, EnvSMTPAddr, "", &k.SMTPAddr, parseHostPort),
		parse(lookup, EnvSMTPUsername, "", &k.SMTPUsername, parseAny),
		parse(lookup, EnvSMTPPassword, "", &k.SMTPPassword, parseOptionalSecret),
		parse(lookup, EnvMailFrom, DefaultMailFrom, &k.MailFrom, parseAny),
		parse(lookup, EnvLoginLinkURL, DefaultLoginLinkURL, &k.LinkURL, parseLinkURL),
		parse(lookup, EnvRateLimitKeyRequest, strconv.Itoa(DefaultRateLimitKeyRequest), &k.RateLimit, parseRateLimit),
		parse(lookup, EnvRateLimitKeyRequestAddress, strconv.Itoa(DefaultRateLimitKeyRequestAddress), &k.RateLimitAddress, parseRateLimit),
	)
	if err != nil || !k.Enabled() {
		return err
	}
	var errs []error
	if k.SMTPUsername == "" {
		errs = append(errs, fmt.Errorf("%w: %s: required with %s", ErrInvalid, EnvSMTPUsername, EnvSMTPAddr))
	}
	if k.SMTPPassword == nil {
		errs = append(errs, fmt.Errorf("%w: %s: required with %s", ErrInvalid, EnvSMTPPassword, EnvSMTPAddr))
	}
	if !d.Enabled() {
		errs = append(errs, fmt.Errorf("%w: %s needs the demo's %s and %s: each key request is proven by Turnstile",
			ErrInvalid, EnvSMTPAddr, EnvDemoOrigin, EnvTurnstileSecret))
	}
	return errors.Join(errs...)
}

// parseAny accepts any string.
func parseAny(s string) (string, error) {
	return s, nil
}

// parseHostPort accepts an empty string, as none, or "host:port" with a
// host and a port from 1 to 65535.
func parseHostPort(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if _, err := parseAddr(s); err != nil {
		return "", err
	}
	if strings.HasPrefix(s, ":") {
		return "", errors.New("a host is required")
	}
	return s, nil
}

// parseLinkURL accepts an absolute HTTPS URL with a host, no credentials,
// no query and no fragment: the token is appended as its fragment.
func parseLinkURL(s string) (string, error) {
	if _, err := parseHTTPSNoQuery(s); err != nil {
		return "", err
	}
	return s, nil
}
