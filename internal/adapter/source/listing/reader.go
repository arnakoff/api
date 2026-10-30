// SPDX-License-Identifier: AGPL-3.0-only

package listing

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arnakoff/api/internal/app"
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
	snapshots Snapshots
	source    domain.Source

	build   sync.Mutex // one indexing at a time
	current atomic.Pointer[index]
}

// index is the blacklist of the snapshot fetched at fetchedAt, or why it
// could not be built.
type index struct {
	fetchedAt time.Time
	list      *domain.Blacklist
	err       error
}

// NewReader returns the Reader of the blacklist of source in snapshots.
func NewReader(snapshots Snapshots, source domain.Source) *Reader {
	return &Reader{snapshots: snapshots, source: source}
}

// Blacklist returns the blacklist in service and when it was fetched,
// app.ErrNoSnapshot when none is, or ErrSnapshot when it is malformed.
func (r *Reader) Blacklist() (*domain.Blacklist, time.Time, error) {
	snap, ok := r.snapshots.Snapshot(r.source)
	if !ok {
		return nil, time.Time{}, fmt.Errorf("blacklist %s: %w", r.source, app.ErrNoSnapshot)
	}
	idx := r.current.Load()
	if idx == nil || !idx.fetchedAt.Equal(snap.FetchedAt) {
		idx = r.index(snap)
	}
	return idx.list, idx.fetchedAt, idx.err
}

// index builds and keeps the index of snap, unless another read did. While
// one read builds it, the others keep the previous index, if any, rather
// than wait.
func (r *Reader) index(snap domain.Snapshot) *index {
	if !r.build.TryLock() {
		if idx := r.current.Load(); idx != nil {
			return idx
		}
		r.build.Lock()
	}
	defer r.build.Unlock()
	if idx := r.current.Load(); idx != nil && idx.fetchedAt.Equal(snap.FetchedAt) {
		return idx
	}
	idx := &index{fetchedAt: snap.FetchedAt}
	idx.list, idx.err = blacklistOf(snap)
	if idx.err != nil {
		idx.err = fmt.Errorf("blacklist %s: %w", r.source, idx.err)
	}
	r.current.Store(idx)
	return idx
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
	listedOn, err := time.Parse(time.DateOnly, row[posListedOn])
	if err != nil {
		return fmt.Errorf("%w: %s not AAAA-MM-JJ", ErrSnapshot, ColumnListedOn)
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
