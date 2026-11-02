// SPDX-License-Identifier: AGPL-3.0-only

package signal_test

import (
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

const (
	numberingURL = "https://www.data.gouv.fr/datasets/ressources-en-numerotation-telephonique"
	spoofing     = "Un numéro affiché peut être usurpé : rappelle l'organisme au numéro que tu trouves toi-même sur son site officiel."
)

// testPlan holds ranges of the ARCEP numbering fixture (testdata/arcep):
// one with its holder's name, one with its code only.
func testPlan(t testing.TB) *domain.NumberingPlan {
	t.Helper()
	sfr, err := domain.NewNumberRange("0612000000", "0612999999", domain.NumberAssignment{
		Prefix: "0612", Operator: "SFR0", OperatorName: "Société française du radiotéléphone", OperatorSIREN: "343059564",
		Territory: "Métropole", AssignedOn: day(2011, time.December, 8),
	})
	if err != nil {
		t.Fatal(err)
	}
	unnamed, err := domain.NewNumberRange("0159140000", "0159149999", domain.NumberAssignment{
		Prefix: "015914", Operator: "VOIC", Territory: "Métropole",
		AssignedOn: day(2022, time.December, 9),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewNumberingPlan([]domain.NumberRange{sfr, unnamed})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestNumberingSignal(t *testing.T) {
	t.Parallel()
	plan := testPlan(t)
	tests := []struct {
		name  string
		phone string
		code  string // "" for no signal
		level domain.Level
		msg   string
	}{
		{
			"assigned range, named holder", "06 12 34 56 78", "phone_operator", domain.LevelInfo,
			"Le numéro appartient à une tranche attribuée à Société française du radiotéléphone (SFR0) le 08/12/2011 (numérotation ARCEP). " +
				"Le numéro a pu être porté chez un autre opérateur.",
		},
		{
			"assigned range, holder known by its code only", "01 59 14 12 34", "phone_operator", domain.LevelInfo,
			"Le numéro appartient à une tranche attribuée à VOIC le 09/12/2022 (numérotation ARCEP). " +
				"Le numéro a pu être porté chez un autre opérateur.",
		},
		{
			"no assigned range", "01 41 00 00 00", "phone_unassigned", domain.LevelAttention,
			"Le numéro n'appartient à aucune tranche attribuée par l'ARCEP : il ne peut pas être en service.",
		},
		{"outside the French plan", "+44 20 7946 0958", "", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := domain.ParsePhone(tt.phone)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := signal.Numbering(p, plan, checkedAt)
			if tt.code == "" {
				if ok {
					t.Fatalf("Numbering() = %+v, want no signal", got)
				}
				return
			}
			want := domain.Signal{
				Code:       tt.code,
				Level:      tt.level,
				Message:    tt.msg,
				Advice:     spoofing,
				Source:     "ARCEP",
				SourceURL:  numberingURL,
				ObservedAt: checkedAt,
			}
			if !ok || got != want {
				t.Errorf("Numbering() = %+v, %v, want %+v", got, ok, want)
			}
		})
	}
}

func BenchmarkNumbering(b *testing.B) {
	plan := testPlan(b)
	p, err := domain.ParsePhone("01 41 00 00 00")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := signal.Numbering(p, plan, checkedAt); !ok {
			b.Fatal("no signal")
		}
	}
}
