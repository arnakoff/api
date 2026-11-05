// SPDX-License-Identifier: AGPL-3.0-only

package objectstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/defaults"
)

// Defaults of KeyCheckOptions. How long a new key of Scaleway IAM takes to
// work on Object Storage is not documented (« à vérifier », ADR 0007): six
// tries a minute apart leave it five minutes.
const (
	defaultKeyCheckAttempts = 6
	defaultKeyCheckPause    = time.Minute
)

// KeyCheckOptions configures a KeyCheck.
type KeyCheckOptions struct {
	// Endpoint is the HTTPS URL of the S3 API; "https://" + Host when
	// empty.
	Endpoint string

	// Bucket is the bucket the key must read. Required.
	Bucket string

	// HTTP sends the requests. Required.
	HTTP s3.HTTPClient

	// Attempts is the number of listings tried, 6 when zero; Pause the
	// wait between two of them, a minute when zero.
	Attempts int
	Pause    time.Duration

	// Sleep waits d, or returns the error of ctx once it is done; the
	// system clock when nil.
	Sleep func(ctx context.Context, d time.Duration) error
}

// KeyCheck checks that a new key reads a bucket before it is put in
// service (ADR 0007). It implements app.CredentialInstaller and changes
// nothing.
type KeyCheck struct {
	opts KeyCheckOptions
}

// NewKeyCheck returns a KeyCheck of opts, or ErrInvalidOptions.
func NewKeyCheck(opts KeyCheckOptions) (*KeyCheck, error) {
	err := errors.Join(
		defaults.Set("attempts", &opts.Attempts, defaultKeyCheckAttempts),
		defaults.Set("pause", &opts.Pause, defaultKeyCheckPause),
	)
	switch {
	case err != nil:
		return nil, fmt.Errorf("%w: %w", ErrInvalidOptions, err)
	case opts.Bucket == "" || opts.HTTP == nil:
		return nil, fmt.Errorf("%w: a bucket and an HTTP client are required", ErrInvalidOptions)
	case opts.Sleep == nil:
		opts.Sleep = clock.System{}.Sleep
	}
	return &KeyCheck{opts: opts}, nil
}

// InstallCredential lists at most one object of the bucket with c, until a
// listing succeeds or the attempts are over.
func (k *KeyCheck) InstallCredential(ctx context.Context, c app.Credential) error {
	client, err := newClient(Options{
		Endpoint:  k.opts.Endpoint,
		Bucket:    k.opts.Bucket,
		AccessKey: c.AccessKey,
		SecretKey: config.Secret(c.SecretKey),
		HTTP:      k.opts.HTTP,
		Attempts:  1,
	})
	if err != nil {
		return err
	}
	for attempt := 1; ; attempt++ {
		if err = k.list(ctx, client); err == nil {
			return nil
		}
		if attempt == k.opts.Attempts || k.opts.Sleep(ctx, k.opts.Pause) != nil {
			return fmt.Errorf("key %s does not read the bucket %s: %w", c.AccessKey, k.opts.Bucket, err)
		}
	}
}

// list lists at most one object of the bucket.
func (k *KeyCheck) list(ctx context.Context, client *s3.Client) error {
	_, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(k.opts.Bucket),
		MaxKeys: aws.Int32(1),
	})
	return err
}
