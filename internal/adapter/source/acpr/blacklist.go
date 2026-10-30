// SPDX-License-Identifier: AGPL-3.0-only

package acpr

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
	"github.com/arnakoff/api/internal/platform/csvfile"
)

// BlacklistSource is the name of the ACPR blacklist, as shown to users.
const BlacklistSource = signal.ACPRSource

// ErrFormat means the file is not laid out as docs/sources.md records it.
// Errors never quote a value of the file: it may hold an address.
var ErrFormat = errors.New("unexpected ACPR file format")

var blacklistColumns = listing.BlacklistColumns()

// blacklistHeader is the header of the published file (docs/sources.md).
var blacklistHeader = []string{"URL / Mails", "Dénomination", "Catégorie(s)", "Date d'inscription"}

// blacklistMinRows is the least number of rows a sane file gives. On
// 30/10/2026 the file gave 11 531 entries and 11 455 rows were kept; the
// margin allows for delistings.
const blacklistMinRows = 10000

// frenchDate is the JJ/MM/AAAA layout of the listing dates.
const frenchDate = "02/01/2006"

// ParseBlacklist reads the published CSV file into a snapshot, one row per
// site or address, with the categories separated by "|" and the date as
// AAAA-MM-JJ, as in the AMF blacklist snapshot. Free-text names and the
// Dénomination column are left out. A file that departs from the recorded
// format is rejected with ErrFormat rather than half read.
func ParseBlacklist(r io.Reader) (domain.Snapshot, error) {
	cr, err := csvfile.OpenSeparated(r, ',', blacklistHeader, ErrFormat)
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
// and moves snap.Newest to its date. Two real entries have no category:
// they are kept with none.
func addListing(snap *domain.Snapshot, rec []string) error {
	entry, categories, listed := rec[0], rec[2], rec[3]
	date, err := time.Parse(frenchDate, listed)
	if err != nil {
		return fmt.Errorf("%w: Date d'inscription not JJ/MM/AAAA", ErrFormat)
	}
	k, ok := listing.Classify(entry)
	if !ok {
		return nil
	}
	snap.Rows = append(snap.Rows, k.Fields(strings.ReplaceAll(categories, ", ", listing.CategorySeparator), date.Format(time.DateOnly)))
	if date.After(snap.Newest) {
		snap.Newest = date
	}
	return nil
}
