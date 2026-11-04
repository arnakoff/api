// SPDX-License-Identifier: AGPL-3.0-only

package amf

import (
	"fmt"

	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/domain"
)

// Positions of the white list columns in a snapshot row, after the four
// common ones.
const (
	posNumber = 4
	posName   = 5

	posKind   = 7
	posStatus = 8
	posFrom   = 9
	posUntil  = 10

	posRegisteredOn = 6
)

// PSANWhitelist indexes a PSAN snapshot, for listing.NewWhitelistReader,
// leaving out the pages of platforms (listing.WhitelistOf). A withdrawn
// authorisation ends on its end date; a running one has none.
func PSANWhitelist(snap domain.Snapshot, platforms domain.Platforms) (*domain.Whitelist, error) {
	return listing.WhitelistOf(snap, psanColumns, psanAuthorisationOf, platforms)
}

// OffersWhitelist indexes a biens divers snapshot, for
// listing.NewWhitelistReader, leaving out the pages of platforms
// (listing.WhitelistOf).
func OffersWhitelist(snap domain.Snapshot, platforms domain.Platforms) (*domain.Whitelist, error) {
	return listing.WhitelistOf(snap, offersColumns, func(row []string) (domain.Authorisation, error) {
		from, err := listing.ParseDate(ColumnRegisteredOn, row[posRegisteredOn])
		return domain.Authorisation{Name: row[posName], Number: row[posNumber], From: from}, err
	}, platforms)
}

// checkWhitelist returns the domain.SnapshotSpec.Rows check of a white list
// that build indexes. Platforms only decide which pages are left out, never
// whether a row reads back, so none are needed.
func checkWhitelist(build func(domain.Snapshot, domain.Platforms) (*domain.Whitelist, error)) func(domain.Snapshot) error {
	return func(snap domain.Snapshot) error {
		_, err := build(snap, domain.Platforms{})
		return err
	}
}

// psanAuthorisationOf reads the authorisation of a PSAN snapshot row.
func psanAuthorisationOf(row []string) (domain.Authorisation, error) {
	a := domain.Authorisation{Name: row[posName], Number: row[posNumber], Kind: row[posKind]}
	var err error
	if a.From, err = listing.ParseDate(ColumnAuthorisedFrom, row[posFrom]); err != nil {
		return domain.Authorisation{}, err
	}
	switch row[posStatus] {
	case StatusAuthorised:
		return a, nil
	case StatusWithdrawn:
		a.Until, err = listing.ParseDate(ColumnAuthorisedUntil, row[posUntil])
		return a, err
	default:
		return domain.Authorisation{}, fmt.Errorf("%w: unknown %s", listing.ErrSnapshot, ColumnStatus)
	}
}
