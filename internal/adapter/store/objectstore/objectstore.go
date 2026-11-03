// SPDX-License-Identifier: AGPL-3.0-only

package objectstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/config"
)

// Scaleway Object Storage in fr-par (ADR 0005), the defaults of Options.
const (
	Host   = "s3.fr-par.scw.cloud"
	Region = "fr-par"
)

// DefaultAttempts is the number of tries of a request the bucket answers
// with a 5xx, a throttle or a network error, the first one included.
const DefaultAttempts = 3

// ErrInvalidOptions means New was given Options it cannot use.
var ErrInvalidOptions = errors.New("invalid object store options")

// Options configures a Store.
type Options struct {
	// Endpoint is the HTTPS URL of the S3 API; "https://" + Host when
	// empty.
	Endpoint string

	// Region signs the requests; Region when empty.
	Region string

	// Bucket holds the snapshots. Required.
	Bucket string

	// AccessKey and SecretKey are the credentials of the bucket. Required.
	AccessKey string
	SecretKey config.Secret

	// HTTP sends the requests: the platform client, which allows the
	// endpoint host only and bounds time and size. Required.
	HTTP s3.HTTPClient

	// Attempts is the number of tries of a request; DefaultAttempts when
	// 0.
	Attempts int
}

// Store keeps one object per source in a bucket. It implements
// app.SnapshotStore and is safe for concurrent use.
type Store struct {
	client *s3.Client
	bucket string

	mu   sync.Mutex
	seen map[domain.Source]seenObject
}

// seenObject is the last object Load accepted for a source.
type seenObject struct {
	etag string
	snap domain.Snapshot
}

// New returns a Store for opts, or ErrInvalidOptions.
func New(opts Options) (*Store, error) {
	opts, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}
	secret := opts.SecretKey
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(opts.Endpoint),
		Region:       opts.Region,
		UsePathStyle: true,
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: opts.AccessKey, SecretAccessKey: string(secret.Bytes())}, nil
		}),
		HTTPClient:       opts.HTTP,
		RetryMaxAttempts: opts.Attempts,
		// The snapshot format carries its own SHA-256 digest; the optional
		// CRC checksums of recent SDKs are not sent nor required, as S3
		// compatible stores do not all support them.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	return &Store{client: client, bucket: opts.Bucket, seen: map[domain.Source]seenObject{}}, nil
}

// withDefaults returns opts with its defaults, or ErrInvalidOptions. Its
// errors never quote a key.
func (opts Options) withDefaults() (Options, error) {
	if opts.Endpoint == "" {
		opts.Endpoint = "https://" + Host
	}
	if opts.Region == "" {
		opts.Region = Region
	}
	if opts.Attempts == 0 {
		opts.Attempts = DefaultAttempts
	}
	u, err := url.Parse(opts.Endpoint)
	switch {
	case err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil:
		return Options{}, fmt.Errorf("%w: endpoint is not an https:// URL with a host", ErrInvalidOptions)
	case opts.Bucket == "":
		return Options{}, fmt.Errorf("%w: no bucket", ErrInvalidOptions)
	case opts.AccessKey == "" || len(opts.SecretKey) == 0:
		return Options{}, fmt.Errorf("%w: access key and secret key required", ErrInvalidOptions)
	case opts.HTTP == nil:
		return Options{}, fmt.Errorf("%w: no HTTP client", ErrInvalidOptions)
	case opts.Attempts < 0:
		return Options{}, fmt.Errorf("%w: negative attempts", ErrInvalidOptions)
	}
	return opts, nil
}

// Save replaces the object of s.Source with s.
func (st *Store) Save(ctx context.Context, s domain.Snapshot) error {
	key, err := store.FileName(s.Source)
	if err != nil {
		return err
	}
	b, err := store.Encode(s)
	if err != nil {
		return fmt.Errorf("save %s: %w", s.Source, err)
	}
	_, err = st.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(st.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(b),
		ContentLength: aws.Int64(int64(len(b))),
	})
	if err != nil {
		return fmt.Errorf("save %s: %w", s.Source, err)
	}
	return nil
}

// Load returns the snapshot in the object of source: app.ErrNoSnapshot
// when there is none, store.ErrCorrupt or store.ErrVersion when it does not
// check out. An object unchanged since the last Load is not downloaded.
func (st *Store) Load(ctx context.Context, source domain.Source) (domain.Snapshot, error) {
	key, err := store.FileName(source)
	if err != nil {
		return domain.Snapshot{}, err
	}
	in := &s3.GetObjectInput{Bucket: aws.String(st.bucket), Key: aws.String(key)}
	prev, known := st.lastSeen(source)
	if known {
		in.IfNoneMatch = aws.String(prev.etag)
	}
	out, err := st.client.GetObject(ctx, in)
	switch status(err) {
	case http.StatusNotModified:
		if known {
			return prev.snap, nil
		}
	case http.StatusNotFound:
		st.forget(source)
		return domain.Snapshot{}, fmt.Errorf("load %s: %w", source, app.ErrNoSnapshot)
	}
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("load %s: %w", source, err)
	}
	return st.accept(source, key, out)
}

// accept reads and checks the object out of source, and remembers it.
func (st *Store) accept(source domain.Source, key string, out *s3.GetObjectOutput) (domain.Snapshot, error) {
	defer func() { _ = out.Body.Close() }() // Read in full; a close error changes nothing.
	b, err := io.ReadAll(out.Body)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("load %s: %w", source, err)
	}
	s, err := store.Decode(b, source)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("load %s from %s: %w", source, key, err)
	}
	if tag := aws.ToString(out.ETag); tag != "" {
		st.remember(source, seenObject{etag: tag, snap: s})
	}
	return s, nil
}

// status returns the HTTP status of an error answer of the bucket, or 0.
func status(err error) int {
	var re interface{ HTTPStatusCode() int }
	if errors.As(err, &re) {
		return re.HTTPStatusCode()
	}
	return 0
}

func (st *Store) lastSeen(source domain.Source) (seenObject, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	o, ok := st.seen[source]
	return o, ok
}

func (st *Store) remember(source domain.Source, o seenObject) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.seen[source] = o
}

func (st *Store) forget(source domain.Source) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.seen, source)
}
