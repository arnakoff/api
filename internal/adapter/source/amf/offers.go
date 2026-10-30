// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"io"
	"strings"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/csvfile"
)

// OffersSource is the name of the AMF white list of offers in goods (biens
// divers: wine, cattle, trees…), as shown to users.
const OffersSource = signal.OffersSource

// OffersDatasetURL is the data.gouv.fr API page of the biens divers dataset.
const OffersDatasetURL = "https://www.data.gouv.fr/api/1/datasets/65709c9cae162fbc34e7ec05/"

// ColumnRegisteredOn is the registration date of an offer, AAAA-MM-JJ.
const ColumnRegisteredOn = "registered_on"

var offersColumns = listing.Columns(ColumnAMFNumber, ColumnName, ColumnRegisteredOn)

// offersHeader is the header of the published file (docs/sources.md).
var offersHeader = []string{"no_amf", "offre_nom", "url_offre", "telephone", "date_enregistrement", "date_de_publication"}

// offersMinRows is the least number of rows a sane file gives. On
// 29/10/2026 the file listed 12 offers, which gave 12 rows: 6 sites and 6
// phone numbers. The 20 % shrink check of the sync guards the rest.
const offersMinRows = 8

// NewOffers returns the Fetcher of the biens divers white list, which reads
// the dataset metadata at datasetURL, OffersDatasetURL in production,
// through client.
func NewOffers(client Getter, datasetURL string) *Fetcher {
	return &Fetcher{
		client:     client,
		datasetURL: datasetURL,
		spec:       domain.SnapshotSpec{Source: OffersSource, Columns: offersColumns, MinRows: offersMinRows},
		parse:      ParseOffers,
	}
}

// ParseOffers reads the published CSV file into a snapshot, one row per
// site and phone number of a registered offer. An offer with neither gives
// no row.
func ParseOffers(r io.Reader) (domain.Snapshot, error) {
	cr, err := csvfile.Open(r, offersHeader, ErrFormat)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snap := domain.Snapshot{Source: OffersSource, Header: offersColumns}
	if err := cr.Each(func(rec []string) error { return addOffer(&snap, rec) }); err != nil {
		return domain.Snapshot{}, err
	}
	return snap, nil
}

// addOffer appends the rows of the offer rec to snap and moves snap.Newest
// to its registration date.
func addOffer(snap *domain.Snapshot, rec []string) error {
	number, name, site, phone, registered := rec[0], strings.TrimSpace(rec[1]), rec[2], rec[3], rec[4]
	if number == "" {
		return errEmpty("no_amf")
	}
	if name == "" {
		return errEmpty("offre_nom")
	}
	date, err := parseDate("date_enregistrement", registered)
	if err != nil {
		return err
	}
	if k, ok := listing.Site(site); ok {
		snap.Rows = append(snap.Rows, k.Fields(number, name, registered))
	}
	if k, ok := listing.Phone(phone); ok {
		snap.Rows = append(snap.Rows, k.Fields(number, name, registered))
	}
	if date.After(snap.Newest) {
		snap.Newest = date
	}
	return nil
}
