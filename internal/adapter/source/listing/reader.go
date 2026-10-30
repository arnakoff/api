// SPDX-License-Identifier: AGPL-3.0-only

package listing

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// ErrSnapshot means a blacklist snapshot departs from the rows the
// connectors write. Errors locate the row, never quote a value: it may hold
// an address.
var ErrSnapshot = errors.New("malformed blacklist snapshot")

// Snapshots returns the snapshot in service for a source; *app.Sync is one.
type Snapshots interface {
	Snapshot(source domain.Source) (domain.Snapshot, bool)
}

// Reader is the app.BlacklistReader of one blacklist. It indexes each
// snapshot in service once, at the first read after it is swapped in, and
// is safe for concurrent use.
type Reader struct {
	cache cache[*domain.Blacklist]
}

// NewReader returns the Reader of the blacklist of source in snapshots.
func NewReader(snapshots Snapshots, source domain.Source) *Reader {
	return &Reader{cache: cache[*domain.Blacklist]{snapshots: snapshots, source: source, build: blacklistOf}}
}

// Blacklist returns the blacklist in service and when it was fetched,
// app.ErrNoSnapshot when none is, or ErrSnapshot when it is malformed.
func (r *Reader) Blacklist() (*domain.Blacklist, time.Time, error) {
	return r.cache.get()
}

// Positions of the blacklist columns in a row.
const (
	posKind = iota
	posKey
	posPath
	posEntry
	posCategories
	posListedOn
	rowLen
)

// blacklistOf indexes the rows of snap. Page entries are left out: a
// checked domain has no path, and a page does not stand for its site.
func blacklistOf(snap domain.Snapshot) (*domain.Blacklist, error) {
	if !slices.Equal(snap.Header, BlacklistColumns()) {
		return nil, fmt.Errorf("%w: unexpected header", ErrSnapshot)
	}
	list := domain.NewBlacklist()
	for i, row := range snap.Rows {
		if err := add(list, row); err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
	}
	return list, nil
}

// add lists the entry of row in list, unless it names a page.
func add(list *domain.Blacklist, row []string) error {
	if len(row) != rowLen {
		return fmt.Errorf("%w: %d fields, %d expected", ErrSnapshot, len(row), rowLen)
	}
	if row[posPath] != "" {
		return nil
	}
	listedOn, err := ParseDate(ColumnListedOn, row[posListedOn])
	if err != nil {
		return err
	}
	l := domain.Listing{ListedOn: listedOn}
	if cats := row[posCategories]; cats != "" {
		l.Categories = strings.Split(cats, CategorySeparator)
	}
	return addEntry(list, row[posKind], row[posKey], l)
}

// addEntry lists the site or address key, of the given kind, in list.
func addEntry(list *domain.Blacklist, kind, key string, l domain.Listing) error {
	switch kind {
	case domain.KindDomain.String():
		d, err := domain.ParseDomain(key)
		if err != nil {
			return fmt.Errorf("%w: invalid site", ErrSnapshot)
		}
		list.AddSite(d, l)
	case domain.KindEmail.String():
		e, err := domain.ParseEmail(key)
		if err != nil {
			return fmt.Errorf("%w: invalid address", ErrSnapshot)
		}
		list.AddEmail(e, l)
	default:
		return fmt.Errorf("%w: kind neither %s nor %s", ErrSnapshot, domain.KindDomain, domain.KindEmail)
	}
	return nil
}

// ParseDate reads the AAAA-MM-JJ date of column in a snapshot row.
func ParseDate(column, s string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s not AAAA-MM-JJ", ErrSnapshot, column)
	}
	return t, nil
}
