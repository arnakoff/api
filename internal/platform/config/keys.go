// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"crypto/subtle"
	"encoding/pem"
	"errors"
	"fmt"
)

// DefaultAPIKeyEnv accepts the production keys, "ark_live_…".
const DefaultAPIKeyEnv = "live"

// RequireKeyStore returns an error, wrapping ErrInvalid, unless the API key
// store is configured: a database and an API key secret other than the log
// key, so that a leaked log key does not also let anyone check a key.
func (c Config) RequireKeyStore() error {
	errs := []error{c.RequireDatabase()}
	switch {
	case len(c.APIKeySecret) == 0:
		errs = append(errs, fmt.Errorf("%w: %s: required, generate one with: openssl rand -hex 32", ErrInvalid, EnvAPIKeySecret))
	case subtle.ConstantTimeCompare(c.APIKeySecret, c.LogKey) == 1:
		errs = append(errs, fmt.Errorf("%w: %s: must differ from %s", ErrInvalid, EnvAPIKeySecret, EnvLogKey))
	}
	return errors.Join(errs...)
}

// RequireDatabase returns an error, wrapping ErrInvalid, unless the
// database of the API key store is set.
func (c Config) RequireDatabase() error {
	if len(c.DatabaseURL) == 0 {
		return fmt.Errorf("%w: %s: required", ErrInvalid, EnvDatabaseURL)
	}
	return nil
}

// parseOptionalSecret accepts any string, an empty one as none.
func parseOptionalSecret(s string) (Secret, error) {
	if s == "" {
		return nil, nil
	}
	return Secret(s), nil
}

// parseOptionalKey accepts an empty string, as none, or a key parseKey
// accepts.
func parseOptionalKey(s string) (Secret, error) {
	if s == "" {
		return nil, nil
	}
	return parseKey(s)
}

// parseOptionalPEM accepts an empty string, as none, or a text holding at
// least one PEM CERTIFICATE block. Whether it parses as a certificate is
// checked where it is used.
func parseOptionalPEM(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	for rest := []byte(s); ; {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return "", errors.New("no PEM CERTIFICATE block")
		}
		if block.Type == "CERTIFICATE" {
			return s, nil
		}
	}
}

// parseAPIKeyEnv accepts "live" or "test".
func parseAPIKeyEnv(s string) (string, error) {
	if s != "live" && s != "test" {
		return "", fmt.Errorf("%q is neither live nor test", s)
	}
	return s, nil
}
