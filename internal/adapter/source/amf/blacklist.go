// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"io"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/csvfile"
)

// BlacklistSource is the name of the AMF blacklist, as shown to users.
const BlacklistSource = signal.AMFSource

// BlacklistDatasetURL is the data.gouv.fr API page of the blacklist dataset.
const BlacklistDatasetURL = "https://www.data.gouv.fr/api/1/datasets/641d5dab1d84b530f7720b08/"

var blacklistColumns = listing.BlacklistColumns()

// blacklistHeader is the header of the published file (docs/sources.md).
var blacklistHeader = []string{"nom", "categorie", "date_inscription"}

// blacklistMinRows is the least number of rows a sane file gives. On
// 29/10/2026 the file gave 3 466 and the list only grows, by 350 to 430
// entries a year; the margin allows for delistings.
const blacklistMinRows = 3000

// NewBlacklist returns the Fetcher of the blacklist, which reads the dataset
// metadata at datasetURL, BlacklistDatasetURL in production, through client.
func NewBlacklist(client Getter, datasetURL string) *Fetcher {
	return &Fetcher{
		client:     client,
		datasetURL: datasetURL,
		spec:       domain.SnapshotSpec{Source: BlacklistSource, Columns: blacklistColumns, MinRows: blacklistMinRows},
		parse:      ParseBlacklist,
	}
}

// ParseBlacklist reads the published CSV file into a snapshot, one row per
// site or address. Free-text names are left out. A file that departs from
// the recorded format is rejected with ErrFormat rather than half read.
func ParseBlacklist(r io.Reader) (domain.Snapshot, error) {
	cr, err := csvfile.Open(r, blacklistHeader, ErrFormat)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snap := domain.Snapshot{Source: BlacklistSource, Header: blacklistColumns}
	if err := cr.Each(func(rec []string) error { return addListing(&snap, rec) }); err != nil {
		return domain.Snapshot{}, err
	}
	return snap, nil
}

// addListing appends the listing rec to snap, unless it is a free-text name,
// and moves snap.Newest to its date.
func addListing(snap *domain.Snapshot, rec []string) error {
	nom, categories, listed := rec[0], rec[1], rec[2]
	if categories == "" {
		return errEmpty("categorie")
	}
	date, err := parseDate("date_inscription", listed)
	if err != nil {
		return err
	}
	k, ok := listing.Classify(nom)
	if !ok {
		return nil
	}
	k.Entry = nom
	snap.Rows = append(snap.Rows, k.Fields(categories, listed))
	if date.After(snap.Newest) {
		snap.Newest = date
	}
	return nil
}
