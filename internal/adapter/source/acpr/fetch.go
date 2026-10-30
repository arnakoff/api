// SPDX-License-Identifier: AGPL-3.0-only

package acpr

import (
	"bytes"
	"context"
	"fmt"

	"github.com/arnakoff/api/internal/domain"
)

// BlacklistURL is the published file of the blacklist.
const BlacklistURL = "https://www.abe-infoservice.fr/fr/abeis-liste-noire.csv"

// Hosts returns the hosts a Fetcher reaches, for the HTTP client allowlist.
func Hosts() []string {
	return []string{"www.abe-infoservice.fr"}
}

// Getter fetches a URL; *httpclient.Client is one.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// Fetcher downloads and parses the blacklist. It implements
// app.SourceFetcher.
type Fetcher struct {
	client  Getter
	fileURL string
}

// NewBlacklist returns the Fetcher of the blacklist, which downloads the
// file at fileURL, BlacklistURL in production, through client.
func NewBlacklist(client Getter, fileURL string) *Fetcher {
	return &Fetcher{client: client, fileURL: fileURL}
}

// Spec declares the snapshot columns and the least number of rows.
func (f *Fetcher) Spec() domain.SnapshotSpec {
	return domain.SnapshotSpec{Source: BlacklistSource, Columns: blacklistColumns, MinRows: blacklistMinRows}
}

// Fetch downloads the file and parses it.
func (f *Fetcher) Fetch(ctx context.Context) (domain.Snapshot, error) {
	body, err := f.client.Get(ctx, f.fileURL)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s file: %w", BlacklistSource, err)
	}
	snap, err := ParseBlacklist(bytes.NewReader(body))
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("%s file: %w", BlacklistSource, err)
	}
	return snap, nil
}
