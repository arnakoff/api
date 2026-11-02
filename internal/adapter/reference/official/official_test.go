// SPDX-License-Identifier: AGPL-3.0-only

package official_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/reference/official"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

var checkedAt = time.Date(2026, time.November, 2, 9, 0, 0, 0, time.UTC)

func lookalike(t *testing.T, o signal.Officials, host string) (domain.Signal, bool) {
	t.Helper()
	d, err := domain.ParseDomain(host)
	if err != nil {
		t.Fatalf("ParseDomain(%q): %v", host, err)
	}
	return o.Lookalike(d, checkedAt)
}

func TestParseReadsEveryColumn(t *testing.T) {
	t.Parallel()

	const csv = "domain,alternates,names,of,proof\n" +
		"credit-agricole.fr,credit-agricole.com,creditagricole|ca-en-ligne,du Crédit Agricole,https://proof.example/ca\n"
	o, err := official.Parse(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for host, want := range map[string]bool{
		"credit-agricole.com":     false, // alternate
		"creditagricole-pro.net":  true,  // name
		"ca-en-ligne.fr":          true,  // name of several words
		"credit-agricole-web.com": false, // a label is no name
	} {
		s, got := lookalike(t, o, host)
		if got != want {
			t.Errorf("Lookalike(%q) = %v; want %v", host, got, want)
		}
		if got && (s.SourceURL != "https://proof.example/ca" || !strings.Contains(s.Message, "site officiel du Crédit Agricole")) {
			t.Errorf("Lookalike(%q) = %+v; want the row's proof and operator", host, s)
		}
	}
}

func TestParseRejectsMalformedFiles(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"other header":   "domain,names,of,proof\nameli.fr,,x,https://p.example/\n",
		"missing column": "domain,alternates,names,of,proof\nameli.fr,,de l'Assurance maladie,https://p.example/\n",
		"invalid row":    "domain,alternates,names,of,proof\nwww.ameli.fr,,,de l'Assurance maladie,https://p.example/\n",
		"empty":          "",
	}
	for name, csv := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := official.Parse(strings.NewReader(csv)); err == nil {
				t.Error("Parse() error = nil; want an error")
			}
		})
	}
	if _, err := official.Parse(strings.NewReader(tests["other header"])); !errors.Is(err, official.ErrFormat) {
		t.Errorf("Parse(other header) error = %v; want ErrFormat", err)
	}
	_, err := official.Parse(strings.NewReader(tests["invalid row"]))
	if !errors.Is(err, signal.ErrInvalidOfficial) {
		t.Errorf("Parse(invalid row) error = %v; want ErrInvalidOfficial", err)
	}
}

// TestLoadReadsTheShippedList checks that the embedded list is valid and
// that its official sites are no look-alikes.
func TestLoadReadsTheShippedList(t *testing.T) {
	t.Parallel()

	o, err := official.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, host := range []string{"www.ameli.fr", "www.impots.gouv.fr", "www.credit-agricole.fr"} {
		if s, found := lookalike(t, o, host); found {
			t.Errorf("Lookalike(%q) = %+v; want no signal", host, s)
		}
	}
	if _, found := lookalike(t, o, "ameli-remboursement.com"); !found {
		t.Error("Lookalike(ameli-remboursement.com) = false; want a signal")
	}
}

func TestLoadListsBNPParibas(t *testing.T) {
	t.Parallel()

	o, err := official.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, tc := range []struct {
		host string
		want bool
	}{
		{"www.bnpparibas.fr", false},             // official domain
		{"bnpparibas.com", false},                // alternate, redirects to the bank
		{"mabanque.bnpparibas", false},           // brand suffix, not listed
		{"mabanque.fr", false},                   // generic label is no name
		{"mabanque-en-ligne.com", false},         // generic label is no name
		{"bnpparibas-securite.com", true},        // name
		{"espace-bnp-paribas.net", true},         // name of several words
		{"bnp-client.fr", true},                  // short name, owner decision
		{"bnpparibaas.fr", true},                 // one edit away from the label
		{"mabanque.bnpparibas-client.com", true}, // name in the registrable domain
	} {
		_, found := lookalike(t, o, tc.host)
		if found != tc.want {
			t.Errorf("Lookalike(%q) found = %t; want %t", tc.host, found, tc.want)
		}
	}
}

// BenchmarkLookalikeShippedList measures the rule on the /check hot path,
// against the whole shipped list, for a host that imitates nothing.
func BenchmarkLookalikeShippedList(b *testing.B) {
	o, err := official.Load()
	if err != nil {
		b.Fatalf("Load: %v", err)
	}
	d, err := domain.ParseDomain("www.boutique-exemple.fr")
	if err != nil {
		b.Fatalf("ParseDomain: %v", err)
	}
	b.ReportAllocs()
	for b.Loop() {
		o.Lookalike(d, checkedAt)
	}
}
