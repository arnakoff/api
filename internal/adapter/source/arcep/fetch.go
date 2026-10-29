// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"bytes"
	"context"
	"fmt"

	"github.com/arnakoff/api/internal/domain"
)

// Hosts returns the hosts a Fetcher reaches, for the HTTP client allowlist.
func Hosts() []string {
	return []string{"extranet.arcep.fr"}
}

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Fetcher downloads the numbering and operators files and joins them into
// a numbering snapshot. It implements app.SourceFetcher.
type Fetcher struct {
	client       Getter
	numberingURL string
	operatorsURL string
}

// NewNumbering returns the Fetcher of the numbering plan, which downloads
// the files at numberingURL and operatorsURL, NumberingURL and OperatorsURL
// in production, through client.
func NewNumbering(client Getter, numberingURL, operatorsURL string) *Fetcher {
	return &Fetcher{client: client, numberingURL: numberingURL, operatorsURL: operatorsURL}
}

// Spec declares the snapshot columns and the least number of rows.
func (f *Fetcher) Spec() domain.SnapshotSpec {
	return domain.SnapshotSpec{Source: NumberingSource, Columns: numberingColumns, MinRows: numberingMinRows}
}

// Fetch downloads and parses the operators file, then the numbering file.
func (f *Fetcher) Fetch(ctx context.Context) (domain.Snapshot, error) {
	body, err := f.client.Get(ctx, f.operatorsURL)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s operators file: %w", NumberingSource, err)
	}
	operators, err := ParseOperators(bytes.NewReader(body))
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s operators file: %w", NumberingSource, err)
	}
	body, err = f.client.Get(ctx, f.numberingURL)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s numbering file: %w", NumberingSource, err)
	}
	snap, err := ParseNumbering(bytes.NewReader(body), operators)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s numbering file: %w", NumberingSource, err)
	}
	return snap, nil
}
