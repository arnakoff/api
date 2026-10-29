// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
)

// fixtures holds trimmed extracts of the real files, downloaded on
// 29/10/2026, in their published Windows-1252 encoding.
var fixtures = os.DirFS("../../../../testdata/arcep")

// Fixtures: 34 ranges of 21 operators, and those 21 operators plus one that
// holds no range.
const (
	numberingFixture = "numbering.csv"
	operatorsFixture = "operators.csv"
)

func readFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	b, err := fs.ReadFile(fixtures, name)
	if err != nil {
		tb.Fatalf("read fixture: %v", err)
	}
	return b
}

// operatorsHeaderLine is the header of the published operators file.
const operatorsHeaderLine = "IDENTITE_OPERATEUR;CODE_OPERATEUR;SIRET_ACTEUR;RCS_ACTEUR;ADRESSE_COMPLETE_ACTEUR;ATTRIB_RESS_NUM;DATE_DECLARATION_OPERATEUR\r\n"

func TestParseOperatorsFixture(t *testing.T) {
	t.Parallel()
	ops, err := ParseOperators(bytes.NewReader(readFixture(t, operatorsFixture)))
	if err != nil {
		t.Fatalf("ParseOperators() error = %v", err)
	}
	if len(ops) != 22 {
		t.Errorf("len(ParseOperators()) = %d, want 22", len(ops))
	}
	want := map[string]Operator{
		"FRTE": {Name: "Orange", SIREN: "380129866"},
		"SFR0": {Name: "Société française du radiotéléphone", SIREN: "343059564"},
		"BUYC": {Name: "Digicel Antilles Françaises Guyane", SIREN: "431416288"},
		"LNUM": {Name: "118 218 Le numéro", SIREN: "478343080"},
		"VOIC": {Name: "8x8 UK Limited"},
	}
	for code, w := range want {
		if got := ops[code]; got != w {
			t.Errorf("ParseOperators()[%s] = %+v, want %+v", code, got, w)
		}
	}
}

func TestParseOperatorsSIREN(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"SIRET in two groups":   "380129866 48625",
		"SIRET in four groups":  "380 129 866 48625",
		"SIREN alone":           "380129866",
		"foreign registry":      "05083841",
		"empty":                 "",
		"SIRET with a bad key":  "380129866 48626",
		"SIREN with a bad key":  "380129867",
		"registry name, no key": "Register of companies",
	}
	want := map[string]string{
		"SIRET in two groups":  "380129866",
		"SIRET in four groups": "380129866",
		"SIREN alone":          "380129866",
	}
	for name, siret := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := operatorsHeaderLine + `Orange;FRTE;"` + siret + `";Paris;"1 RUE X";1;12/03/1998` + "\r\n"
			ops, err := ParseOperators(strings.NewReader(in))
			if err != nil {
				t.Fatalf("ParseOperators() error = %v", err)
			}
			if got := ops["FRTE"].SIREN; got != want[name] {
				t.Errorf("SIREN = %q, want %q", got, want[name])
			}
		})
	}
}

func TestParseOperatorsErrors(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"empty":          "",
		"other header":   "IDENTITE_OPERATEUR;CODE_OPERATEUR\r\n",
		"empty name":     operatorsHeaderLine + `"";FRTE;;;;1;12/03/1998` + "\r\n",
		"empty code":     operatorsHeaderLine + `Orange;;;;;1;12/03/1998` + "\r\n",
		"duplicate code": operatorsHeaderLine + "Orange;FRTE;;;;1;12/03/1998\r\nOrange SA;FRTE;;;;1;12/03/1998\r\n",
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseOperators(strings.NewReader(in)); !errors.Is(err, ErrFormat) {
				t.Errorf("ParseOperators() error = %v, want %v", err, ErrFormat)
			}
		})
	}
}
