// SPDX-License-Identifier: AGPL-3.0-only

package signaldoc_test

import (
	"errors"
	"os"
	"testing"

	"github.com/arnakoff/api/internal/adapter/signaldoc"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

func TestTableListsEachDefinitionAsARow(t *testing.T) {
	t.Parallel()

	defs := []signal.Definition{
		{
			Code: "amf_blacklist", Level: domain.LevelAlert, Inputs: []domain.Kind{domain.KindDomain, domain.KindEmail},
			Source: "AMF", Condition: "Site inscrit",
		},
		{Code: "x", Level: domain.LevelInfo, Inputs: []domain.Kind{domain.KindIBAN}, Source: "IBAN", Condition: "a | b"},
	}

	want := "| Code | Entrées | Niveau | Source | Condition |\n" +
		"|---|---|---|---|---|\n" +
		"| `amf_blacklist` | domain, email | alerte | AMF | Site inscrit |\n" +
		"| `x` | iban | info | IBAN | a \\| b |\n"
	if got := signaldoc.Table(defs); got != want {
		t.Errorf("Table() =\n%s\nwant\n%s", got, want)
	}
}

func TestRewriteReplacesWhatTheMarkersHold(t *testing.T) {
	t.Parallel()

	doc := "# Signaux\n\n" + signaldoc.Begin + "\nold\n" + signaldoc.End + "\n\n## Suite\n"

	got, err := signaldoc.Rewrite(doc, "new\n")
	if err != nil {
		t.Fatal(err)
	}
	if want := "# Signaux\n\n" + signaldoc.Begin + "\nnew\n" + signaldoc.End + "\n\n## Suite\n"; got != want {
		t.Errorf("Rewrite() =\n%s\nwant\n%s", got, want)
	}
}

func TestRewriteNeedsBothMarkersInOrder(t *testing.T) {
	t.Parallel()

	for name, doc := range map[string]string{
		"none":     "# Signaux\n",
		"no end":   signaldoc.Begin + "\n",
		"no begin": signaldoc.End + "\n",
		"reversed": signaldoc.End + "\n" + signaldoc.Begin + "\n",
	} {
		if _, err := signaldoc.Rewrite(doc, "new\n"); !errors.Is(err, signaldoc.ErrMarkers) {
			t.Errorf("Rewrite(%s) error = %v, want ErrMarkers", name, err)
		}
	}
}

// docs/signals.md is written by go generate: a change to the catalog not
// regenerated fails here.
func TestSignalsDocumentIsUpToDate(t *testing.T) {
	t.Parallel()

	doc, err := os.ReadFile("../../../docs/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	want, err := signaldoc.Rewrite(string(doc), signaldoc.Table(signal.Catalog()))
	if err != nil {
		t.Fatal(err)
	}
	if string(doc) != want {
		t.Error("docs/signals.md is not up to date with the catalog: run go generate ./...")
	}
}
