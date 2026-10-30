// SPDX-License-Identifier: AGPL-3.0-only

package corpus_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/corpus"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

const header = "input,type,label,added_at,note\n"

func TestReadParsesEachCase(t *testing.T) {
	t.Parallel()

	file := header +
		"https://www.exemple.fr/contact,domain,scam,2026-10-08,liste noire AMF : Usurpation\n" +
		"contact@exemple.fr,email,scam,2026-10-06,\"liste noire ACPR : Crédits, Usurpation\"\n" +
		"120027016,siren,legit,2026-10-30,AMF PSAN\n"

	got, err := corpus.Read(strings.NewReader(file))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := []struct {
		in    string
		kind  domain.Kind
		label app.Label
	}{
		{"www.exemple.fr", domain.KindDomain, app.LabelScam},
		{"contact@exemple.fr", domain.KindEmail, app.LabelScam},
		{"120027016", domain.KindSIREN, app.LabelLegit},
	}
	if len(got) != len(want) {
		t.Fatalf("Read = %d cases, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Input.String() != w.in || got[i].Input.Kind() != w.kind || got[i].Label != w.label {
			t.Errorf("case %d = %v %v %v, want %v %v %v", i,
				got[i].Input, got[i].Input.Kind(), got[i].Label, w.in, w.kind, w.label)
		}
	}
}

func TestReadRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
	}{
		{"wrong header", "input,type,label\n"},
		{"unknown type", header + "exemple.fr,url,scam,2026-10-08,\n"},
		{"auto type", header + "exemple.fr,auto,scam,2026-10-08,\n"},
		{"unknown label", header + "exemple.fr,domain,suspect,2026-10-08,\n"},
		{"input of another type", header + "exemple.fr,iban,scam,2026-10-08,\n"},
		{"bad date", header + "exemple.fr,domain,scam,08/10/2026,\n"},
		{"missing field", header + "exemple.fr,domain,scam\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := corpus.Read(strings.NewReader(tt.file))
			if !errors.Is(err, corpus.ErrFormat) {
				t.Errorf("Read error = %v, want ErrFormat", err)
			}
		})
	}
}

func TestLoadReadsBothFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write(t, dir, "scam.csv", header+"armen.ltd,domain,scam,2026-10-08,\n")
	write(t, dir, "legit.csv", header+"www.impots.gouv.fr,domain,legit,2026-10-30,\n")

	got, err := corpus.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 || got[0].Label != app.LabelScam || got[1].Label != app.LabelLegit {
		t.Errorf("Load = %+v, want the scam case then the legit one", got)
	}
}

func TestLoadRejectsALabelInTheWrongFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write(t, dir, "scam.csv", header+"www.impots.gouv.fr,domain,legit,2026-10-30,\n")
	write(t, dir, "legit.csv", header)

	if _, err := corpus.Load(dir); !errors.Is(err, corpus.ErrFormat) {
		t.Errorf("Load error = %v, want ErrFormat", err)
	}
}

func TestLoadRejectsAMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write(t, dir, "scam.csv", header)

	if _, err := corpus.Load(dir); err == nil {
		t.Error("Load without legit.csv: want an error")
	}
}

// The committed corpus reads: every row has a known type, label and date.
func TestTheCorpusReads(t *testing.T) {
	t.Parallel()

	got, err := corpus.Load(filepath.Join("..", "..", "..", "testdata", "corpus"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) == 0 {
		t.Error("the corpus is empty")
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
