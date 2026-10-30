// SPDX-License-Identifier: AGPL-3.0-only

package corpus_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/corpus"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

func TestWriteReport(t *testing.T) {
	t.Parallel()

	s := app.Score{
		Scams: 185, ScamsAlerted: 180, ScamsFlagged: 182,
		Legit: 293, LegitAlerted: 0,
		FalseAlerts: []app.FalseAlert{
			{Input: "www.impots.gouv.fr", Level: domain.LevelAttention},
			{Input: "lcl.fr", Level: domain.LevelAttention},
		},
		Unavailable: 3,
	}
	var b strings.Builder

	if err := corpus.WriteReport(&b, s, []string{"AMF", "ACPR"}); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}

	want := "Sources left out: AMF, ACPR.\n\n" +
		"| | this run |\n" +
		"|---|---|\n" +
		"| scams with ≥ 1 `alerte` | 180 / 185 |\n" +
		"| scams with ≥ 1 `attention` | 182 / 185 |\n" +
		"| legit with ≥ 1 `attention` (false alerts) | 2 / 293 |\n" +
		"| legit with ≥ 1 `alerte` (must stay 0) | 0 / 293 |\n" +
		"| cases with a source unavailable | 3 / 478 |\n" +
		"\nFalse alerts:\n\n" +
		"- `www.impots.gouv.fr` (attention)\n" +
		"- `lcl.fr` (attention)\n"
	if b.String() != want {
		t.Errorf("WriteReport wrote\n%s\nwant\n%s", b.String(), want)
	}
}

func TestWriteReportWithAllSourcesAndNoFalseAlert(t *testing.T) {
	t.Parallel()

	var b strings.Builder

	if err := corpus.WriteReport(&b, app.Score{Scams: 1, Legit: 1}, nil); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}

	got := b.String()
	if !strings.HasPrefix(got, "Sources left out: none.\n") || strings.Contains(got, "False alerts") {
		t.Errorf("WriteReport wrote\n%s", got)
	}
}

func TestWriteReportReturnsTheWriteError(t *testing.T) {
	t.Parallel()

	if err := corpus.WriteReport(failingWriter{}, app.Score{}, nil); !errors.Is(err, errWrite) {
		t.Errorf("WriteReport error = %v, want the write error", err)
	}
}

var errWrite = errors.New("disk full")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }
