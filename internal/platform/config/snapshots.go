// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
)

// Environment variables of the snapshot bucket, read by Load.
const (
	EnvSnapshotBucket          = "ARNAKOFF_SNAPSHOT_BUCKET"
	EnvSnapshotBucketAccessKey = "ARNAKOFF_SNAPSHOT_BUCKET_ACCESS_KEY"
	EnvSnapshotBucketSecretKey = "ARNAKOFF_SNAPSHOT_BUCKET_SECRET_KEY" //nolint:gosec // G101: the name of a variable, not its value.
)

// Bounds of a bucket name.
const (
	minBucketName = 3
	maxBucketName = 63
)

// SnapshotBucket is the S3 bucket the snapshots are stored in, in
// production (ADR 0005); off when its name is empty, and then they stay in
// SnapshotDir.
type SnapshotBucket struct {
	// Name is the bucket: 3 to 63 lower-case letters, digits and dashes,
	// starting and ending with a letter or a digit.
	Name string

	// AccessKey and SecretKey are the credentials of the bucket: the key
	// pair of an IAM application restricted to it. Both are Secrets, kept
	// out of every log.
	AccessKey Secret
	SecretKey Secret
}

// Enabled reports whether the snapshots are stored in the bucket.
func (b SnapshotBucket) Enabled() bool {
	return b.Name != ""
}

// parseSnapshotBucket sets *b from the variables of the bucket, read with
// lookup. The name and the two keys are set together or not at all.
func parseSnapshotBucket(lookup func(string) (string, bool), b *SnapshotBucket) error {
	return parseBucket(lookup, [3]string{EnvSnapshotBucket, EnvSnapshotBucketAccessKey, EnvSnapshotBucketSecretKey},
		&b.Name, &b.AccessKey, &b.SecretKey)
}

// parseBucket sets *name, *access and *secret from the variables vars, a
// bucket name and its two keys, read with lookup: set together or not at
// all.
func parseBucket(lookup func(string) (string, bool), vars [3]string, name *string, access, secret *Secret) error {
	err := errors.Join(
		parse(lookup, vars[0], "", name, parseBucketName),
		parse(lookup, vars[1], "", access, parseOptionalSecret),
		parse(lookup, vars[2], "", secret, parseOptionalSecret),
	)
	if err != nil {
		return err
	}
	set := *name != ""
	if set != (*access != nil) || set != (*secret != nil) {
		return fmt.Errorf("%w: %s, %s and %s: set all three, or none", ErrInvalid, vars[0], vars[1], vars[2])
	}
	return nil
}

// parseBucketName accepts an empty string, as no bucket, or a name of
// minBucketName to maxBucketName lower-case letters, digits and dashes,
// starting and ending with a letter or a digit: within the S3 naming
// rules, and never a path.
func parseBucketName(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if len(s) < minBucketName || len(s) > maxBucketName {
		return "", fmt.Errorf("%q is not %d to %d characters long", s, minBucketName, maxBucketName)
	}
	for i, r := range s {
		alnum := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		edge := i == 0 || i == len(s)-1
		if !alnum && (edge || r != '-') {
			return "", fmt.Errorf("%q is not lower-case letters, digits and inner dashes", s)
		}
	}
	return s, nil
}
