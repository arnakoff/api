// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
)

// EnvEraseEmail is the address of the account admin account erase
// -listen erases in production (#304), read by Load.
const EnvEraseEmail = "ARNAKOFF_ERASE_EMAIL"

// Erase is the account admin account erase erases.
type Erase struct {
	// Email is the address of the account: personal data, a Secret so
	// that no log shows it; its format is checked where it is used.
	Email Secret
}

// RequireErase returns an error, wrapping ErrInvalid and naming each
// missing variable, unless the database and the address to erase are
// configured.
func (c Config) RequireErase() error {
	errs := []error{c.RequireDatabase()}
	if len(c.Erase.Email) == 0 {
		errs = append(errs, fmt.Errorf("%w: %s: required, the address of the account to erase", ErrInvalid, EnvEraseEmail))
	}
	return errors.Join(errs...)
}

// parseErase sets *e from the variables of the erasure, read with lookup.
func parseErase(lookup func(string) (string, bool), e *Erase) error {
	return parse(lookup, EnvEraseEmail, "", &e.Email, parseOptionalSecret)
}
