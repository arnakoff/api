// SPDX-License-Identifier: AGPL-3.0-only

package catalog_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/amf"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestWhitelists(t *testing.T) {
	t.Parallel()

	var got []domain.Source
	for _, w := range catalog.Whitelists(noSnapshot{}, domain.Platforms{}) {
		got = append(got, w.Rule.Source())
		// Each reader reads its rule's source: none is stored here.
		if _, _, err := w.Reader.Whitelist(); !errors.Is(err, app.ErrNoSnapshot) {
			t.Errorf("%s reader error = %v, want ErrNoSnapshot", w.Rule.Source(), err)
		}
	}
	if want := []domain.Source{"AMF PSAN", "AMF biens divers"}; !slices.Equal(got, want) {
		t.Errorf("whitelists = %v, want %v", got, want)
	}
}

// psanSnapshot serves a PSAN snapshot of one provider, listed under a page
// of linktr.ee.
type psanSnapshot struct{}

func (psanSnapshot) Snapshot(source domain.Source) (domain.Snapshot, bool) {
	if source != amf.PSANSource {
		return domain.Snapshot{}, false
	}
	return domain.Snapshot{Source: source, Header: listing.Columns(
		amf.ColumnAMFNumber, amf.ColumnName, amf.ColumnCountry, amf.ColumnAuthorisation,
		amf.ColumnStatus, amf.ColumnAuthorisedFrom, amf.ColumnAuthorisedUntil,
	), Rows: [][]string{{
		"domain", "linktr.ee", "/exemple", "https://linktr.ee/exemple",
		"E2020-001", "EXEMPLE SAS", "France", "Enregistrement", "Agréé", "2020-03-17", "",
	}}}, true
}

func TestWhitelistsLeaveOutThePagesOfPlatforms(t *testing.T) {
	t.Parallel()

	platforms, err := domain.NewPlatforms([]string{"linktr.ee"})
	if err != nil {
		t.Fatalf("NewPlatforms() error = %v", err)
	}
	linktree, err := domain.ParseDomain("linktr.ee")
	if err != nil {
		t.Fatal(err)
	}
	for guarded, want := range map[bool]int{false: 1, true: 0} {
		p := domain.Platforms{}
		if guarded {
			p = platforms
		}
		psan := catalog.Whitelists(psanSnapshot{}, p)[0]
		w, _, err := psan.Reader.Whitelist()
		if err != nil {
			t.Fatalf("Whitelist() error = %v", err)
		}
		if got := len(w.Authorisations(linktree)); got != want {
			t.Errorf("platforms %v: %d authorisations of linktr.ee, want %d", guarded, got, want)
		}
	}
}
