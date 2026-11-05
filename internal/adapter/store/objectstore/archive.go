// SPDX-License-Identifier: AGPL-3.0-only

package objectstore

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Archive adds the sealed backups of the key store to a bucket out of
// Scaleway (#75). Its credentials may only add objects: it never reads,
// replaces or removes one. It implements app.BackupArchive.
type Archive struct {
	client *s3.Client
	bucket string
}

// NewArchive returns an Archive for opts, or ErrInvalidOptions.
func NewArchive(opts Options) (*Archive, error) {
	client, err := newClient(opts)
	if err != nil {
		return nil, err
	}
	return &Archive{client: client, bucket: opts.Bucket}, nil
}

// Keep adds body as the object name. If-None-Match: * makes the bucket
// refuse a name already taken, so that a backup never replaces another.
func (a *Archive) Keep(ctx context.Context, name string, body []byte) error {
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(a.bucket),
		Key:           aws.String(name),
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
		ContentType:   aws.String("application/octet-stream"),
		IfNoneMatch:   aws.String("*"),
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", name, err)
	}
	return nil
}
