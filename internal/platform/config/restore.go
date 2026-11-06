// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
)

// Environment variables of the restore of the key store in production
// (#297), read by Load.
const (
	EnvRestoreURL      = "ARNAKOFF_RESTORE_URL"
	EnvRestoreIdentity = "ARNAKOFF_RESTORE_IDENTITY"
	EnvRestoreVerify   = "ARNAKOFF_RESTORE_VERIFY_KEY"
)

// Restore is the dump admin restore -listen loads: a presigned URL of the
// dump, sealed for a key drawn for this restore, and that key. Both are
// Secrets, kept out of every log; their format is checked where they are
// used. RequireRestore tells whether it is complete.
type Restore struct {
	// URL is the presigned URL of the sealed dump: its query is a
	// credential.
	URL Secret

	// Identity is the age private key the dump is sealed for, drawn for
	// this restore only.
	Identity Secret

	// VerifyKey is the hex Ed25519 public key that checks the signature of
	// the dump, made by the backup job. It is public: no Secret.
	VerifyKey string
}

// RequireRestore returns an error, wrapping ErrInvalid and naming each
// missing variable, unless the database and the restore are configured.
func (c Config) RequireRestore() error {
	errs := []error{c.RequireDatabase()}
	if len(c.Restore.URL) == 0 {
		errs = append(errs, fmt.Errorf("%w: %s: required, the presigned URL of the sealed dump", ErrInvalid, EnvRestoreURL))
	}
	if len(c.Restore.Identity) == 0 {
		errs = append(errs, fmt.Errorf("%w: %s: required, the age key the dump is sealed for", ErrInvalid, EnvRestoreIdentity))
	}
	return errors.Join(append(errs, c.requireVerifyKey())...)
}

// RequireFileRestore returns an error, wrapping ErrInvalid and naming each
// missing variable, unless the database and the verify key of a clear
// dump are configured.
func (c Config) RequireFileRestore() error {
	return errors.Join(c.RequireDatabase(), c.requireVerifyKey())
}

// requireVerifyKey returns an error unless the verify key is set.
func (c Config) requireVerifyKey() error {
	if c.Restore.VerifyKey == "" {
		return fmt.Errorf("%w: %s: required, the public key that checks the dump signature", ErrInvalid, EnvRestoreVerify)
	}
	return nil
}

// parseRestore sets *r from the variables of the restore, read with
// lookup.
func parseRestore(lookup func(string) (string, bool), r *Restore) error {
	return errors.Join(
		parse(lookup, EnvRestoreURL, "", &r.URL, parseOptionalSecret),
		parse(lookup, EnvRestoreIdentity, "", &r.Identity, parseOptionalSecret),
		parse(lookup, EnvRestoreVerify, "", &r.VerifyKey, parseAny),
	)
}
