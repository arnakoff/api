// SPDX-License-Identifier: AGPL-3.0-only

package listing

import (
	"fmt"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// WhitelistReader is the app.WhitelistReader of one white list. It indexes
// each snapshot in service once, at the first read after it is swapped in,
// and is safe for concurrent use.
type WhitelistReader struct {
	cache cache[*domain.Whitelist]
}

// NewWhitelistReader returns the WhitelistReader of the white list of
// source in snapshots, which build indexes.
func NewWhitelistReader(snapshots Snapshots, source domain.Source, build func(domain.Snapshot) (*domain.Whitelist, error)) *WhitelistReader {
	return &WhitelistReader{cache: cache[*domain.Whitelist]{snapshots: snapshots, source: source, build: build}}
}

// Whitelist returns the white list in service and when it was fetched,
// app.ErrNoSnapshot when none is, or the error of its build.
func (r *WhitelistReader) Whitelist() (*domain.Whitelist, time.Time, error) {
	return r.cache.get()
}

// WhitelistOf indexes the rows of snap, whose header must be header, each
// under the authorisation decode reads from it. A site entry with a path
// lists its whole site: a white list gives the provider's own site, often
// at its home page in French ("/fr/").
func WhitelistOf(snap domain.Snapshot, header []string, decode func(row []string) (domain.Authorisation, error)) (*domain.Whitelist, error) {
	if !slices.Equal(snap.Header, header) {
		return nil, fmt.Errorf("%w: unexpected header", ErrSnapshot)
	}
	w := domain.NewWhitelist()
	for i, row := range snap.Rows {
		if err := addAuthorisation(w, row, len(header), decode); err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
	}
	return w, nil
}

// addAuthorisation lists the key of row, which has n fields, in w.
func addAuthorisation(w *domain.Whitelist, row []string, n int, decode func([]string) (domain.Authorisation, error)) error {
	if len(row) != n {
		return fmt.Errorf("%w: %d fields, %d expected", ErrSnapshot, len(row), n)
	}
	a, err := decode(row)
	if err != nil {
		return err
	}
	return addKey(w, row[posKind], row[posKey], a)
}

// addKey lists the key of the given kind in w.
func addKey(w *domain.Whitelist, kind, key string, a domain.Authorisation) error {
	switch kind {
	case domain.KindDomain.String():
		return addParsed(key, domain.ParseDomain, a, w.AddSite)
	case domain.KindEmail.String():
		return addParsed(key, domain.ParseEmail, a, w.AddEmail)
	case domain.KindPhone.String():
		return addParsed(key, domain.ParsePhone, a, w.AddPhone)
	case domain.KindSIREN.String():
		return addParsed(key, domain.ParseSIREN, a, w.AddSIREN)
	default:
		return fmt.Errorf("%w: unknown kind", ErrSnapshot)
	}
}

// addParsed parses key and adds it with a.
func addParsed[T any](key string, parse func(string) (T, error), a domain.Authorisation, add func(T, domain.Authorisation)) error {
	v, err := parse(key)
	if err != nil {
		return fmt.Errorf("%w: invalid key", ErrSnapshot)
	}
	add(v, a)
	return nil
}
