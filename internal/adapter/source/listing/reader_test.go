// SPDX-License-Identifier: AGPL-3.0-only

package listing_test

import (
	"errors"
	"io"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/source/acpr"
	"github.com/arnakoff/api/internal/adapter/source/amf"
	"github.com/arnakoff/api/internal/adapter/source/listing"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

var fetchedAt = time.Date(2026, time.October, 30, 8, 42, 26, 0, time.UTC)

// snapshots serves one snapshot per source; it is safe for concurrent use.
type snapshots struct {
	mu    sync.Mutex
	snaps map[domain.Source]domain.Snapshot
}

func (s *snapshots) Snapshot(source domain.Source) (domain.Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.snaps[source]
	return snap, ok
}

func (s *snapshots) put(snap domain.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snaps == nil {
		s.snaps = map[domain.Source]domain.Snapshot{}
	}
	s.snaps[snap.Source] = snap
}

var fixtures = os.DirFS("../../../../testdata")

// fixture returns the snapshot parse gives of a blacklist fixture.
func fixture(t testing.TB, path string, parse func(io.Reader) (domain.Snapshot, error)) domain.Snapshot {
	t.Helper()
	f, err := fixtures.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	snap, err := parse(f)
	if err != nil {
		t.Fatal(err)
	}
	snap.FetchedAt = fetchedAt
	return snap
}

func amfSnapshot(t testing.TB) domain.Snapshot {
	return fixture(t, "amf/listesnoires.csv", amf.ParseBlacklist)
}

func acprSnapshot(t testing.TB) domain.Snapshot {
	return fixture(t, "acpr/liste-noire.csv", acpr.ParseBlacklist)
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestReaderIndexesTheSnapshotInService(t *testing.T) {
	t.Parallel()

	var s snapshots
	s.put(amfSnapshot(t))
	s.put(acprSnapshot(t))
	tests := []struct {
		source domain.Source
		in     string
		want   []domain.Listing
	}{
		{amf.BlacklistSource, "armen.ltd", []domain.Listing{{Categories: []string{"Usurpation"}, ListedOn: day(2026, time.October, 8)}}},
		{amf.BlacklistSource, "https://fr.circuit-btc.com/", []domain.Listing{{Categories: []string{"Produits_dérivés_sur_crypto-actifs"}, ListedOn: day(2020, time.September, 10)}}},
		{amf.BlacklistSource, "gestion.finance", []domain.Listing{
			{Categories: []string{"Biens_divers/Or"}, ListedOn: day(2025, time.November, 5)},
			{Categories: []string{"Usurpation"}, ListedOn: day(2025, time.October, 7)},
		}},
		{amf.BlacklistSource, "alphaairobot.com", []domain.Listing{{Categories: []string{"Crypto-actifs", "Usurpation", "Usurpation_AMF"}, ListedOn: day(2026, time.July, 29)}}},
		{amf.BlacklistSource, "support@dbf-link.com", []domain.Listing{{Categories: []string{"Usurpation"}, ListedOn: day(2023, time.August, 23)}}},
		// Page entries name one page, not their site.
		{amf.BlacklistSource, "calendly.com", nil},
		{amf.BlacklistSource, "vantagemarkets.com", nil},
		// « optimalduck.autos/? » targets the whole site.
		{amf.BlacklistSource, "optimalduck.autos", []domain.Listing{{Categories: []string{"Crypto-actifs"}, ListedOn: day(2026, time.September, 10)}}},
		{acpr.BlacklistSource, "monplantfraise.fr", []domain.Listing{{ListedOn: day(2024, time.September, 30)}}},
		{acpr.BlacklistSource, "banque-stellantis.com", []domain.Listing{{Categories: []string{"Crédits livrets paiements assurances", "Usurpation Professionnel"}, ListedOn: day(2026, time.October, 26)}}},
	}
	for _, tt := range tests {
		t.Run(string(tt.source)+" "+tt.in, func(t *testing.T) {
			t.Parallel()

			list, at, err := listing.NewReader(&s, tt.source).Blacklist()
			if err != nil {
				t.Fatalf("Blacklist() error = %v", err)
			}
			if !at.Equal(fetchedAt) {
				t.Errorf("Blacklist() fetched at %v, want %v", at, fetchedAt)
			}
			in, err := domain.ParseInput(tt.in, domain.KindAuto)
			if err != nil {
				t.Fatal(err)
			}
			if got := list.Listings(in); !slices.EqualFunc(got, tt.want, equalListing) {
				t.Errorf("Listings(%s) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestReaderWithoutSnapshot(t *testing.T) {
	t.Parallel()

	_, _, err := listing.NewReader(&snapshots{}, amf.BlacklistSource).Blacklist()
	if !errors.Is(err, app.ErrNoSnapshot) {
		t.Errorf("Blacklist() error = %v, want ErrNoSnapshot", err)
	}
}

func TestReaderIndexesEachSnapshotOnce(t *testing.T) {
	t.Parallel()

	var s snapshots
	s.put(amfSnapshot(t))
	r := listing.NewReader(&s, amf.BlacklistSource)
	first, _, _ := r.Blacklist()
	same, _, _ := r.Blacklist()
	if first != same {
		t.Error("Blacklist() indexed the same snapshot twice")
	}

	next := amfSnapshot(t)
	next.FetchedAt = fetchedAt.Add(time.Hour)
	next.Rows = next.Rows[:1]
	s.put(next)
	list, at, err := r.Blacklist()
	if err != nil || !at.Equal(next.FetchedAt) || list == first {
		t.Fatalf("Blacklist() after a new snapshot = %v, %v, want the new one indexed", at, err)
	}
	if got := list.Listings(domain.Input(mustDomain(t, "armen.ltd"))); len(got) != 0 {
		t.Errorf("Listings(armen.ltd) = %v, want none in the new snapshot", got)
	}
}

func TestReaderRejectsAMalformedSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		edit func(*domain.Snapshot)
	}{
		{"missing column", func(s *domain.Snapshot) { s.Header = s.Header[:len(s.Header)-1] }},
		{"bad date", func(s *domain.Snapshot) {
			s.Rows = [][]string{{"domain", "armen.ltd", "", "armen.ltd", "Forex", "08/10/2026"}}
		}},
		{"bad site", func(s *domain.Snapshot) {
			s.Rows = [][]string{{"domain", "not a host", "", "x", "Forex", "2026-10-08"}}
		}},
		{"bad address", func(s *domain.Snapshot) { s.Rows = [][]string{{"email", "nobody", "", "x", "Forex", "2026-10-08"}} }},
		{"unknown kind", func(s *domain.Snapshot) { s.Rows = [][]string{{"fax", "x", "", "x", "Forex", "2026-10-08"}} }},
		{"short row", func(s *domain.Snapshot) { s.Rows = [][]string{{"domain", "armen.ltd"}} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			snap := amfSnapshot(t)
			tt.edit(&snap)
			var s snapshots
			s.put(snap)
			_, _, err := listing.NewReader(&s, amf.BlacklistSource).Blacklist()
			if !errors.Is(err, listing.ErrSnapshot) {
				t.Errorf("Blacklist() error = %v, want ErrSnapshot", err)
			}
		})
	}
}

func mustDomain(t testing.TB, s string) domain.Domain {
	t.Helper()
	d, err := domain.ParseDomain(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func equalListing(a, b domain.Listing) bool {
	return slices.Equal(a.Categories, b.Categories) && a.ListedOn.Equal(b.ListedOn)
}

func BenchmarkReaderBlacklist(b *testing.B) {
	var s snapshots
	s.put(amfSnapshot(b))
	r := listing.NewReader(&s, amf.BlacklistSource)
	in := domain.Input(mustDomain(b, "www.exemple.fr"))
	b.ReportAllocs()
	for b.Loop() {
		list, _, err := r.Blacklist()
		if err != nil {
			b.Fatal(err)
		}
		list.Listings(in)
	}
}
