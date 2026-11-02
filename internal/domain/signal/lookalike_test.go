// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"errors"
	"testing"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// Rows of the reference list (internal/adapter/reference/official), as
// verified there.
var (
	ameli = signal.Official{
		Domain: "ameli.fr",
		Names:  []string{"ameli"},
		Of:     "de l'Assurance Maladie",
		Proof:  ameliProof,
	}
	creditAgricole = signal.Official{
		Domain:     "credit-agricole.fr",
		Alternates: []string{"credit-agricole.com"},
		Names:      []string{"credit-agricole", "creditagricole"},
		Of:         "de Crédit Agricole S.A.",
		Proof:      agricoleProof,
	}
	laPoste = signal.Official{
		Domain:     "laposte.fr",
		Alternates: []string{"laposte.com"},
		Names:      []string{"laposte", "la-poste"},
		Of:         "de La Poste",
		Proof:      laPosteProof,
	}
	cic = signal.Official{
		Domain: "cic.fr",
		Of:     "du Crédit Industriel et Commercial",
		Proof:  cicProof,
	}
)

// Proof pages of the rows above, verified on 02/11/2026.
const (
	ameliProof    = "https://www.ameli.fr/assure/mentions-legales-cgu"
	agricoleProof = "https://www.credit-agricole.fr/particulier/informations/mentions-legales.html"
	laPosteProof  = "https://www.laposte.fr/mentions-legales"
	cicProof      = "https://www.cic.fr/fr/informations-legales/mentions-legales.html"
)

const lookalikeAdvice = "Tape toi-même l'adresse officielle plutôt que de suivre un lien reçu."

func officials(t testing.TB) signal.Officials {
	t.Helper()
	o, err := signal.NewOfficials([]signal.Official{ameli, creditAgricole, laPoste, cic})
	if err != nil {
		t.Fatalf("NewOfficials: %v", err)
	}
	return o
}

func TestDomainLookalikeSignal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want signal.Official // zero for no signal
	}{
		{"official domain", "ameli.fr", signal.Official{}},
		{"official subdomain", "www.ameli.fr", signal.Official{}},
		{"official alternate", "www.credit-agricole.com", signal.Official{}},
		{"unrelated", "exemple.fr", signal.Official{}},
		{"same label, other suffix", "ameli.com", ameli},
		{"Cyrillic a", "аmeli.fr", ameli},
		{"rn for m", "arneli.fr", ameli},
		{"digit 1 for l", "ame1i.fr", ameli},
		{"digit 0 for o", "lap0ste.fr", laPoste},
		{"one letter missing", "lapste.net", laPoste},
		{"one letter changed", "laposre.fr", laPoste},
		{"two letters swapped", "lapsote.fr", laPoste},
		{"one letter added", "lapostee.fr", laPoste},
		{"two edits", "lapsotee.fr", signal.Official{}},
		{"one edit, under 6 letters", "amelie.fr", signal.Official{}},
		{"name as a word", "ameli-remboursement.com", ameli},
		{"name in a subdomain", "ameli.fr.securite-compte.com", ameli},
		{"name of several words", "www.credit-agricole-securite.com", creditAgricole},
		{"listed spelling of a name", "creditagricole-en-ligne.net", creditAgricole},
		{"look-alike name as a word", "arneli-remboursement.com", ameli},
		{"name inside a word", "cameliaflowers.fr", signal.Official{}},
		{"part of a name", "credit-mutuel-agricole.fr", signal.Official{}},
		{"public suffix only after the name", "ameli.github.io", ameli},
		{"short label, other suffix", "cic.com", signal.Official{}},
		{"short label as a word", "cic-campus.fr", signal.Official{}},
		{"short label, Cyrillic c", "\u0441ic.fr", cic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := officials(t).Lookalike(parseDomain(t, tt.in), checkedAt)
			if tt.want.Domain == "" {
				if ok {
					t.Fatalf("Lookalike(%q) = %+v; want no signal", tt.in, got)
				}
				return
			}
			want := domain.Signal{
				Code:       "domain_lookalike",
				Level:      domain.LevelAttention,
				Message:    "Le domaine " + parseDomain(t, tt.in).String() + " ressemble à " + tt.want.Domain + ", site officiel " + tt.want.Of + ", mais n'en fait pas partie.",
				Advice:     lookalikeAdvice,
				Source:     "Domaines officiels",
				SourceURL:  tt.want.Proof,
				ObservedAt: checkedAt,
			}
			if !ok || got != want {
				t.Errorf("Lookalike(%q) = %+v, %v; want %+v, true", tt.in, got, ok, want)
			}
		})
	}
}

func TestDomainLookalikeMessage(t *testing.T) {
	t.Parallel()

	got, _ := officials(t).Lookalike(parseDomain(t, "ameli-remboursement.com"), checkedAt)
	const want = "Le domaine ameli-remboursement.com ressemble à ameli.fr, site officiel de l'Assurance Maladie, mais n'en fait pas partie."
	if got.Message != want {
		t.Errorf("Message = %q; want %q", got.Message, want)
	}
}

func TestNewOfficialsRejectsInvalidRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  signal.Official
	}{
		{"subdomain", signal.Official{Domain: "www.ameli.fr", Of: "de l'Assurance Maladie", Proof: ameliProof}},
		{"invalid domain", signal.Official{Domain: "ameli", Of: "de l'Assurance Maladie", Proof: ameliProof}},
		{"invalid alternate", signal.Official{Domain: "ameli.fr", Alternates: []string{"x.ameli.com"}, Of: "de l'Assurance Maladie", Proof: ameliProof}},
		{"name not lower-case words", signal.Official{Domain: "ameli.fr", Names: []string{"Ameli Santé"}, Of: "de l'Assurance Maladie", Proof: ameliProof}},
		{"no organisation", signal.Official{Domain: "ameli.fr", Proof: ameliProof}},
		{"no proof", signal.Official{Domain: "ameli.fr", Of: "de l'Assurance Maladie"}},
		{"proof not https", signal.Official{Domain: "ameli.fr", Of: "de l'Assurance Maladie", Proof: "http://www.ameli.fr/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := signal.NewOfficials([]signal.Official{tt.row})
			if !errors.Is(err, signal.ErrInvalidOfficial) {
				t.Errorf("NewOfficials(%+v) error = %v; want ErrInvalidOfficial", tt.row, err)
			}
		})
	}
}

func TestNewOfficialsRejectsDuplicates(t *testing.T) {
	t.Parallel()

	_, err := signal.NewOfficials([]signal.Official{ameli, {Domain: "laposte.fr", Alternates: []string{"ameli.fr"}, Of: "de La Poste", Proof: laPosteProof}})
	if !errors.Is(err, signal.ErrInvalidOfficial) {
		t.Errorf("NewOfficials(duplicate) error = %v; want ErrInvalidOfficial", err)
	}
}

func BenchmarkDomainLookalike(b *testing.B) {
	o := officials(b)
	d := parseDomain(b, "www.boutique-exemple.fr")
	b.ReportAllocs()
	for b.Loop() {
		o.Lookalike(d, checkedAt)
	}
}

func parseDomain(t testing.TB, s string) domain.Domain {
	t.Helper()
	d, err := domain.ParseDomain(s)
	if err != nil {
		t.Fatalf("ParseDomain(%q): %v", s, err)
	}
	return d
}

// FuzzDomainLookalike checks that any domain a user may paste is matched
// without a panic, and that a signal always cites a listed official site.
func FuzzDomainLookalike(f *testing.F) {
	for _, s := range []string{"ameli-remboursement.com", "аmeli.fr", "lapsote.fr", "a.b.c.d.e.cic.fr", "xn--80ak6aa92e.com"} {
		f.Add(s)
	}
	o := officials(f)
	f.Fuzz(func(t *testing.T, s string) {
		d, err := domain.ParseDomain(s)
		if err != nil {
			return
		}
		got, ok := o.Lookalike(d, checkedAt)
		if ok && got.SourceURL != ameliProof && got.SourceURL != agricoleProof && got.SourceURL != laPosteProof && got.SourceURL != cicProof {
			t.Errorf("Lookalike(%q) cites %q", s, got.SourceURL)
		}
	})
}
