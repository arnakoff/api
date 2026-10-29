// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/arnakoff/api/internal/domain"
)

// Hosts returns the hosts a Fetcher reaches, for the HTTP client allowlist:
// the data.gouv.fr API and the object store that serves the files.
func Hosts() []string {
	return []string{"www.data.gouv.fr", "object-api.infra.data.gouv.fr"}
}

// ErrResource means the dataset metadata does not name exactly one CSV file.
var ErrResource = errors.New("no single AMF CSV resource")

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Fetcher downloads and parses one AMF list. It implements
// app.SourceFetcher.
type Fetcher struct {
	client     Getter
	datasetURL string
	spec       domain.SnapshotSpec
	parse      func(io.Reader) (domain.Snapshot, error)
}

// Spec declares the snapshot columns and the least number of rows.
func (f *Fetcher) Spec() domain.SnapshotSpec {
	return f.spec
}

// Fetch resolves the current file from the dataset metadata, downloads it
// and parses it.
func (f *Fetcher) Fetch(ctx context.Context) (domain.Snapshot, error) {
	fileURL, err := f.fileURL(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	body, err := f.client.Get(ctx, fileURL)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s file: %w", f.spec.Source, err)
	}
	snap, err := f.parse(bytes.NewReader(body))
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s file: %w", f.spec.Source, err)
	}
	return snap, nil
}

// dataset holds the fields read from the data.gouv.fr dataset API.
type dataset struct {
	Resources []struct {
		Type   string `json:"type"`
		Format string `json:"format"`
		URL    string `json:"url"`
	} `json:"resources"`
}

// fileURL returns the URL of the dataset's single main CSV resource: the
// AMF dates its file names, so they change with every export.
func (f *Fetcher) fileURL(ctx context.Context) (string, error) {
	body, err := f.client.Get(ctx, f.datasetURL)
	if err != nil {
		return "", fmt.Errorf("%s dataset: %w", f.spec.Source, err)
	}
	var d dataset
	if err := json.Unmarshal(body, &d); err != nil {
		return "", fmt.Errorf("%s dataset: %w: %w", f.spec.Source, ErrResource, err)
	}
	var urls []string
	for _, r := range d.Resources {
		if r.Type == "main" && r.Format == "csv" {
			urls = append(urls, r.URL)
		}
	}
	if len(urls) != 1 {
		return "", fmt.Errorf("%s dataset: %w: %d found", f.spec.Source, ErrResource, len(urls))
	}
	return urls[0], nil
}
