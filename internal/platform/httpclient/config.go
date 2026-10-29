// SPDX-License-Identifier: AGPL-3.0-only

package httpclient

import (
	"crypto/x509"
	"fmt"
	"time"
)

// DefaultUserAgent identifies Arnakoff to upstreams and points to the
// project, so an operator who sees the traffic knows whom to contact.
const DefaultUserAgent = "arnakoff (+https://github.com/arnakoff/api)"

// Defaults applied by New to a zero Config field. They are choices, sized
// for the files and registries listed in docs/sources.md (the largest file,
// ARCEP MAJNUM, is about 1.1 MB).
const (
	DefaultTimeout               = 30 * time.Second
	DefaultDialTimeout           = 5 * time.Second
	DefaultTLSHandshakeTimeout   = 5 * time.Second
	DefaultResponseHeaderTimeout = 10 * time.Second
	DefaultMaxBodyBytes          = 16 << 20
	DefaultAttempts              = 3
	DefaultBaseBackoff           = 500 * time.Millisecond
	DefaultMaxBackoff            = 30 * time.Second
)

// Config configures a Client. Every zero field but AllowedHosts and RootCAs
// takes its Default value; a negative one is invalid.
type Config struct {
	// AllowedHosts are the exact hosts the client may reach, as in URL.Host:
	// "rdap.nic.fr", or "127.0.0.1:8443" with a port. Required.
	AllowedHosts []string

	// UserAgent is sent with every request.
	UserAgent string

	// Timeout bounds a whole Get call: every attempt, every pause between
	// attempts and the reading of the body.
	Timeout time.Duration

	// DialTimeout bounds opening a TCP connection.
	DialTimeout time.Duration

	// TLSHandshakeTimeout bounds the TLS handshake.
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout bounds the wait for response headers once the
	// request is sent.
	ResponseHeaderTimeout time.Duration

	// MaxBodyBytes is the largest response body Get accepts.
	MaxBodyBytes int64

	// Attempts is the number of tries for a 429 or 5xx answer, the first
	// one included.
	Attempts int

	// BaseBackoff is the ceiling of the first pause between two attempts;
	// the ceiling doubles at each retry, up to MaxBackoff.
	BaseBackoff time.Duration

	// MaxBackoff caps a pause. A Retry-After longer than MaxBackoff ends
	// the retries rather than hammer the upstream sooner than it asked.
	MaxBackoff time.Duration

	// RootCAs are the trusted certificate authorities; nil means the
	// system pool.
	RootCAs *x509.CertPool
}

// withDefaults returns cfg with its zero fields set to their Default value,
// or ErrInvalidConfig if a field is negative.
func (cfg Config) withDefaults() (Config, error) {
	durations := []struct {
		name  string
		field *time.Duration
		def   time.Duration
	}{
		{"timeout", &cfg.Timeout, DefaultTimeout},
		{"dial timeout", &cfg.DialTimeout, DefaultDialTimeout},
		{"TLS handshake timeout", &cfg.TLSHandshakeTimeout, DefaultTLSHandshakeTimeout},
		{"response header timeout", &cfg.ResponseHeaderTimeout, DefaultResponseHeaderTimeout},
		{"base backoff", &cfg.BaseBackoff, DefaultBaseBackoff},
		{"max backoff", &cfg.MaxBackoff, DefaultMaxBackoff},
	}
	for _, d := range durations {
		if err := orDefault(d.name, d.field, d.def); err != nil {
			return Config{}, err
		}
	}
	if err := orDefault("max body bytes", &cfg.MaxBodyBytes, DefaultMaxBodyBytes); err != nil {
		return Config{}, err
	}
	if err := orDefault("attempts", &cfg.Attempts, DefaultAttempts); err != nil {
		return Config{}, err
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	return cfg, nil
}

// orDefault sets a zero *field to def and rejects a negative one.
func orDefault[T int | int64 | time.Duration](name string, field *T, def T) error {
	switch {
	case *field < 0:
		return fmt.Errorf("%w: negative %s %v", ErrInvalidConfig, name, *field)
	case *field == 0:
		*field = def
	}
	return nil
}
