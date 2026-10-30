// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"io"
	"slices"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/csvfile"
)

// PSANSource is the name of the AMF white list of digital asset service
// providers (PSAN and MiCA providers), as shown to users.
const PSANSource = signal.PSANSource

// PSANDatasetURL is the data.gouv.fr API page of the PSAN dataset.
const PSANDatasetURL = "https://www.data.gouv.fr/api/1/datasets/653135cf052c9a87787413c2/"

// Columns of a PSAN row, after the common ones.
const (
	// ColumnCountry is the country of the provider's head office, in French.
	ColumnCountry = "country"
	// ColumnAuthorisation is the kind of authorisation ("Enregistrement",
	// "Agrément MICA", "Agrément Pacte").
	ColumnAuthorisation = "authorisation"
	// ColumnStatus is "Agréé" or "Radié".
	ColumnStatus = "status"
	// ColumnAuthorisedFrom is the start date of the authorisation, AAAA-MM-JJ.
	ColumnAuthorisedFrom = "authorised_from"
	// ColumnAuthorisedUntil is its end date, "" while it runs.
	ColumnAuthorisedUntil = "authorised_until"
)

var psanColumns = listing.Columns(
	ColumnAMFNumber, ColumnName, ColumnCountry, ColumnAuthorisation, ColumnStatus, ColumnAuthorisedFrom, ColumnAuthorisedUntil,
)

// psanHeader is the header of the published file (docs/sources.md).
var psanHeader = []string{
	"no_amf", "entite_nom", "forme_juridique", "pays_siege", "site_internet", "email", "telephone",
	"no_registre_national", "lei", "nature_autorisation", "date_debut_autorisation", "date_fin_autorisation",
	"motif_fin_autorisation", "statut", "libelle_type_activite", "libelle_activite", "date_debut_activite",
	"date_fin_activite", "date_de_publication",
}

// Fields of a PSAN record read by the parser, by position in psanHeader.
const (
	psanAMFNumber = 0
	psanName      = 1
	psanCountry   = 3
	psanSite      = 4
	psanEmail     = 5
	psanPhone     = 6
	psanRegistry  = 7
	psanNature    = 9
	psanFrom      = 10
	psanUntil     = 11
	psanStatus    = 13
)

// Statuses of a PSAN authorisation.
const (
	// StatusAuthorised is the status of a running authorisation.
	StatusAuthorised = "Agréé"
	// StatusWithdrawn is the status of an authorisation that ended; its row
	// gives the end date.
	StatusWithdrawn = "Radié"
)

// psanStatuses are the statuses the file uses.
var psanStatuses = []string{StatusAuthorised, StatusWithdrawn}

// psanMinRows is the least number of rows a sane file gives. On 29/10/2026
// the file gave 607: withdrawn providers stay listed as "Radié", so the
// list rarely shrinks.
const psanMinRows = 500

// NewPSAN returns the Fetcher of the PSAN white list, which reads the
// dataset metadata at datasetURL, PSANDatasetURL in production, through
// client.
func NewPSAN(client Getter, datasetURL string) *Fetcher {
	return &Fetcher{
		client:     client,
		datasetURL: datasetURL,
		spec:       domain.SnapshotSpec{Source: PSANSource, Columns: psanColumns, MinRows: psanMinRows},
		parse:      ParsePSAN,
	}
}

// ParsePSAN reads the published CSV file into a snapshot, one row per key
// of a provider's authorisation: each site, address, phone number and, for
// a French provider, SIREN. The file has one record per activity; records
// that give the same row merge. A contact page in the e-mail column and a
// foreign registry number are not keys and are left out.
func ParsePSAN(r io.Reader) (domain.Snapshot, error) {
	cr, err := csvfile.Open(r, psanHeader, ErrFormat)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snap := domain.Snapshot{Source: PSANSource, Header: psanColumns}
	seen := make(map[string]bool)
	err = cr.Each(func(rec []string) error {
		rows, from, err := psanRows(rec)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if id := strings.Join(row, "\x00"); !seen[id] {
				seen[id] = true
				snap.Rows = append(snap.Rows, row)
			}
		}
		if from.After(snap.Newest) {
			snap.Newest = from
		}
		return nil
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return snap, nil
}

// psanRows returns the rows of the record rec, one per key, and the start
// date of its authorisation.
func psanRows(rec []string) ([][]string, time.Time, error) {
	authorisation, from, err := psanAuthorisation(rec)
	if err != nil {
		return nil, time.Time{}, err
	}
	var rows [][]string
	for _, k := range psanKeys(rec) {
		rows = append(rows, k.Fields(authorisation...))
	}
	return rows, from, nil
}

// psanAuthorisation checks and returns the authorisation fields of rec, in
// the order of psanColumns, and its start date.
func psanAuthorisation(rec []string) ([]string, time.Time, error) {
	for _, i := range []int{psanAMFNumber, psanName, psanNature} {
		if strings.TrimSpace(rec[i]) == "" {
			return nil, time.Time{}, errEmpty(psanHeader[i])
		}
	}
	if !slices.Contains(psanStatuses, rec[psanStatus]) {
		return nil, time.Time{}, errUnknown(psanHeader[psanStatus])
	}
	from, err := parseDate(psanHeader[psanFrom], rec[psanFrom])
	if err != nil {
		return nil, time.Time{}, err
	}
	switch until := rec[psanUntil]; {
	case until != "":
		if _, err := parseDate(psanHeader[psanUntil], until); err != nil {
			return nil, time.Time{}, err
		}
	case rec[psanStatus] == StatusWithdrawn:
		return nil, time.Time{}, errEmpty(psanHeader[psanUntil])
	}
	return []string{
		rec[psanAMFNumber], strings.TrimSpace(rec[psanName]), rec[psanCountry],
		rec[psanNature], rec[psanStatus], rec[psanFrom], rec[psanUntil],
	}, from, nil
}

// psanKeys returns the keys of rec. A site column may list several sites,
// separated by spaces.
func psanKeys(rec []string) []listing.Key {
	var keys []listing.Key
	add := func(k listing.Key, ok bool) {
		if ok {
			keys = append(keys, k)
		}
	}
	for _, site := range strings.Fields(rec[psanSite]) {
		add(listing.Site(site))
	}
	add(listing.Email(rec[psanEmail]))
	add(listing.Phone(rec[psanPhone]))
	// Only a French head office has a SIREN; other registries' numbers
	// could pass for one.
	if rec[psanCountry] == "France" {
		add(listing.SIREN(rec[psanRegistry]))
	}
	return keys
}
