// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/arnakoff/api/internal/domain"
)

// DatasetURL is the data.gouv.fr API page of the dataset, which names the
// current file: its URL is dated and changes with every export.
const DatasetURL = "https://www.data.gouv.fr/api/1/datasets/641d5dab1d84b530f7720b08/"

// Hosts returns the hosts a Fetcher reaches, for the HTTP client allowlist:
// the data.gouv.fr API and the object store that serves the file.
func Hosts() []string {
	return []string{"www.data.gouv.fr", "object-api.infra.data.gouv.fr"}
}

// minRows is the least number of rows a sane file gives. On 29/10/2026 the
// file gave 3 466 and the list only grows, by 350 to 430 entries a year;
// the margin allows for delistings.
const minRows = 3000

// ErrResource means the dataset metadata does not name exactly one CSV file.
var ErrResource = errors.New("no single AMF CSV resource")

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Fetcher downloads and parses the AMF blacklist. It implements
// app.SourceFetcher.
type Fetcher struct {
	client     Getter
	datasetURL string
}

// NewFetcher returns a Fetcher that reads the dataset metadata at
// datasetURL, DatasetURL in production, through client.
func NewFetcher(client Getter, datasetURL string) *Fetcher {
	return &Fetcher{client: client, datasetURL: datasetURL}
}

// Spec declares the snapshot columns and the least number of rows.
func (f *Fetcher) Spec() domain.SnapshotSpec {
	return domain.SnapshotSpec{Source: Source, Columns: columns, MinRows: minRows}
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
		return domain.Snapshot{}, fmt.Errorf("amf file: %w", err)
	}
	snap, err := Parse(bytes.NewReader(body))
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("amf file: %w", err)
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

// fileURL returns the URL of the dataset's single main CSV resource.
func (f *Fetcher) fileURL(ctx context.Context) (string, error) {
	body, err := f.client.Get(ctx, f.datasetURL)
	if err != nil {
		return "", fmt.Errorf("amf dataset: %w", err)
	}
	var d dataset
	if err := json.Unmarshal(body, &d); err != nil {
		return "", fmt.Errorf("amf dataset: %w: %w", ErrResource, err)
	}
	var urls []string
	for _, r := range d.Resources {
		if r.Type == "main" && r.Format == "csv" {
			urls = append(urls, r.URL)
		}
	}
	if len(urls) != 1 {
		return "", fmt.Errorf("amf dataset: %w: %d found", ErrResource, len(urls))
	}
	return urls[0], nil
}
