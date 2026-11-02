// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

var (
	amfInfo   = domain.SourceInfo{Name: "AMF", Publisher: "AMF", Licence: "LO 2.0", URL: "https://amf.example"}
	acprInfo  = domain.SourceInfo{Name: "ACPR", Publisher: "ACPR", Licence: "?", URL: "https://acpr.example"}
	rdapInfo  = domain.SourceInfo{Name: "RDAP", Publisher: "IANA", Licence: "-", URL: "https://iana.example"}
	sireneInf = domain.SourceInfo{Name: "SIRENE", Publisher: "INSEE", Licence: "LO 2.0", URL: "https://insee.example"}
)

func TestSourceListShowsSyncedSourcesInSyncOrderThenTheOthers(t *testing.T) {
	t.Parallel()

	s, _ := newSync(t, newStore(),
		newFetcher(amfSpec).then(snapshot(100, t0), nil), newFetcher(acprSpec))
	if err := s.RunSource(t.Context(), "AMF"); err != nil {
		t.Fatalf("RunSource: %v", err)
	}
	l, err := app.NewSourceList(s, []domain.SourceInfo{acprInfo, amfInfo}, rdapInfo, sireneInf)
	if err != nil {
		t.Fatalf("NewSourceList: %v", err)
	}

	want := []app.ListedSource{
		{Info: amfInfo, Synced: true, Sync: app.SourceSync{Source: "AMF", LastSuccess: t0, Rows: 100, Newest: t0}},
		{Info: acprInfo, Synced: true, Sync: app.SourceSync{Source: "ACPR"}},
		{Info: rdapInfo},
		{Info: sireneInf},
	}
	if got := l.Sources(); !slices.Equal(got, want) {
		t.Errorf("Sources() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestNewSourceListRefusesAMismatchWithTheSync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		synced []domain.SourceInfo
		want   error
	}{
		{"a synced source without info", []domain.SourceInfo{amfInfo}, app.ErrUndescribedSource},
		{"an info of no synced source", []domain.SourceInfo{amfInfo, acprInfo, rdapInfo}, app.ErrUnknownSource},
		{"a source described twice", []domain.SourceInfo{amfInfo, acprInfo, amfInfo}, app.ErrDuplicateSource},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, _ := newSync(t, newStore(), newFetcher(amfSpec), newFetcher(acprSpec))

			if _, err := app.NewSourceList(s, tt.synced); !errors.Is(err, tt.want) {
				t.Errorf("NewSourceList() error = %v, want %v", err, tt.want)
			}
		})
	}
}
