// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

var fixtures = os.DirFS("../../../../testdata/bodacc")

func readFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	b, err := fs.ReadFile(fixtures, name)
	if err != nil {
		tb.Fatalf("read fixture: %v", err)
	}
	return b
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestParseNotices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fixture string
		siren   string
		want    []domain.Notice
	}{
		{
			name:    "no notice",
			fixture: "no-notice.json",
			siren:   "120027016",
			want:    []domain.Notice{},
		},
		{
			name:    "procedures then radiation",
			fixture: "radiation.json",
			siren:   "504352360",
			want: []domain.Notice{
				{ID: "B2026020947", Kind: domain.NoticeRadiation, Published: day(2026, time.October, 30)},
				{ID: "A201700991916", Kind: domain.NoticeCollectiveProcedure, Published: day(2017, time.May, 24)},
				{ID: "A201700471420", Kind: domain.NoticeCollectiveProcedure, Published: day(2017, time.March, 8)},
				{ID: "A201601051406", Kind: domain.NoticeCollectiveProcedure, Published: day(2016, time.May, 29)},
			},
		},
		{
			name:    "conciliation, one of a group's notices",
			fixture: "conciliation.json",
			siren:   "802618793",
			want: []domain.Notice{
				{ID: "A202602002579", Kind: domain.NoticeConciliation, Published: day(2026, time.October, 18)},
				{ID: "A202302002551", Kind: domain.NoticeConciliation, Published: day(2023, time.October, 17)},
			},
		},
		{
			// A202500814208 cancels A202500322546: both are left out.
			name:    "cancelled notice",
			fixture: "annulation.json",
			siren:   "835084732",
			want: []domain.Notice{
				{ID: "A202600102443", Kind: domain.NoticeCollectiveProcedure, Published: day(2026, time.January, 16)},
				{ID: "A202501362944", Kind: domain.NoticeCollectiveProcedure, Published: day(2025, time.July, 18)},
				{ID: "A202400132157", Kind: domain.NoticeCollectiveProcedure, Published: day(2024, time.January, 19)},
			},
		},
		{
			// A202602093931 corrects A202601993983 and takes its place.
			name:    "corrected notice",
			fixture: "rectificatif.json",
			siren:   "402175558",
			want: []domain.Notice{
				{ID: "A202602093931", Kind: domain.NoticeCollectiveProcedure, Published: day(2026, time.October, 30)},
			},
		},
		{
			name:    "notices about another company are skipped",
			fixture: "radiation.json",
			siren:   "835084732",
			want:    []domain.Notice{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseNotices(readFixture(t, tt.fixture), tt.siren)
			if err != nil {
				t.Fatalf("parseNotices() error = %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseNotices() = %v, want %v", got, tt.want)
			}
		})
	}
}

// record returns an answer with one notice of family and type, pointing
// to precedent: raw JSON, null or a string holding the precedent's JSON.
func record(family, typ, precedent string) []byte {
	return fmt.Appendf(nil, `{"results":[{"id":"A202602093931","registre":["504352360","504 352 360"],`+
		`"dateparution":"2026-10-30","familleavis":%q,"typeavis":%q,"publicationavis":"A",`+
		`"parution":"20260209","numeroannonce":3931,"parutionavisprecedent":%s}]}`, family, typ, precedent)
}

// precedent returns the parutionavisprecedent of a notice that points to
// notice number of parution in publication.
func precedent(publication, parution, number string) string {
	return strconv.Quote(fmt.Sprintf(`{"nomPublication": %q, "numeroParution": %q, "dateParution": "2026-10-16", "numeroAnnonce": %q}`,
		publication, parution, number))
}

func TestParseNoticesKinds(t *testing.T) {
	t.Parallel()
	tests := map[string]domain.NoticeKind{
		"collective":                   domain.NoticeCollectiveProcedure,
		"radiation":                    domain.NoticeRadiation,
		"conciliation":                 domain.NoticeConciliation,
		"retablissement_professionnel": domain.NoticeProfessionalRecovery,
	}
	for family, want := range tests {
		t.Run(family, func(t *testing.T) {
			t.Parallel()
			got, err := parseNotices(record(family, "annonce", "null"), "504352360")
			if err != nil {
				t.Fatalf("parseNotices() error = %v", err)
			}
			if len(got) != 1 || got[0].Kind != want {
				t.Errorf("parseNotices() = %v, want one notice of kind %d", got, want)
			}
		})
	}
}

func TestParseNoticesInvalid(t *testing.T) {
	t.Parallel()
	tests := map[string][]byte{
		"not JSON":                      []byte(`<html>`),
		"no results field":              []byte(`{"total_count":0}`),
		"no id":                         []byte(`{"results":[{"registre":["504352360"],"dateparution":"2026-10-30","familleavis":"collective","typeavis":"annonce"}]}`),
		"bad date":                      []byte(`{"results":[{"id":"A1","registre":["504352360"],"dateparution":"30/10/2026","familleavis":"collective","typeavis":"annonce"}]}`),
		"family not asked for":          record("dpc", "annonce", "null"),
		"unknown type":                  record("collective", "autre", "null"),
		"correction without precedent":  record("collective", "rectificatif", "null"),
		"precedent not JSON":            record("collective", "annulation", `"{"`),
		"precedent unknown publication": record("collective", "annulation", precedent("JORF", "20260199", "3983")),
		"precedent bad number":          record("collective", "annulation", precedent("BODACC A", "20260199", "x")),
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseNotices(body, "504352360"); !errors.Is(err, ErrResponse) {
				t.Errorf("parseNotices() error = %v, want ErrResponse", err)
			}
		})
	}
}
