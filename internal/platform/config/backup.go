// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
)

// Environment variables of the backup of the key store (#75), read by
// Load.
const (
	EnvBackupBucket          = "ARNAKOFF_BACKUP_BUCKET"
	EnvBackupBucketAccessKey = "ARNAKOFF_BACKUP_BUCKET_ACCESS_KEY"
	EnvBackupBucketSecretKey = "ARNAKOFF_BACKUP_BUCKET_SECRET_KEY" //nolint:gosec // G101: the name of a variable, not its value.
	EnvBackupRecipient       = "ARNAKOFF_BACKUP_RECIPIENT"
)

// Backup is where admin backup keeps the sealed copies of the key store:
// a bucket out of Scaleway, and the public key they are sealed for. Every
// field is empty when unset; RequireBackup tells whether it is complete.
type Backup struct {
	// Bucket is the bucket name, as SnapshotBucket.Name.
	Bucket string

	// AccessKey and SecretKey are the credentials of the bucket, which may
	// only add objects. Both are Secrets, kept out of every log.
	AccessKey Secret
	SecretKey Secret

	// Recipient is the age public key the backups are sealed for. It is
	// public: no Secret. Its format is checked where it is used.
	Recipient string
}

// RequireBackup returns an error, wrapping ErrInvalid and naming each
// missing variable, unless the database and the backup are configured.
func (c Config) RequireBackup() error {
	errs := []error{c.RequireDatabase()}
	if c.Backup.Bucket == "" {
		errs = append(errs, fmt.Errorf("%w: %s, %s and %s: required", ErrInvalid,
			EnvBackupBucket, EnvBackupBucketAccessKey, EnvBackupBucketSecretKey))
	}
	if c.Backup.Recipient == "" {
		errs = append(errs, fmt.Errorf("%w: %s: required, the age public key of the owner", ErrInvalid, EnvBackupRecipient))
	}
	return errors.Join(errs...)
}

// parseBackup sets *b from the variables of the backup, read with lookup.
// The bucket and its two keys are set together or not at all.
func parseBackup(lookup func(string) (string, bool), b *Backup) error {
	return errors.Join(
		parseBucket(lookup, [3]string{EnvBackupBucket, EnvBackupBucketAccessKey, EnvBackupBucketSecretKey},
			&b.Bucket, &b.AccessKey, &b.SecretKey),
		parse(lookup, EnvBackupRecipient, "", &b.Recipient, parseAny),
	)
}
