// SPDX-License-Identifier: AGPL-3.0-only

package corpus

import (
	"fmt"
	"io"
	"strings"

	"github.com/arnakoff/api/internal/app"
)

// WriteReport writes s as the Markdown table of the eval-signals skill,
// naming the sources the run left out, then lists each false alert. The
// inputs listed are legit ones: no personal data.
func WriteReport(w io.Writer, s app.Score, without []string) error {
	left := "none"
	if len(without) > 0 {
		left = strings.Join(without, ", ")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Sources left out: %s.\n\n", left)
	b.WriteString("| | this run |\n|---|---|\n")
	row := func(label string, n, of int) { fmt.Fprintf(&b, "| %s | %d / %d |\n", label, n, of) }
	row("scams with ≥ 1 `alerte`", s.ScamsAlerted, s.Scams)
	row("scams with ≥ 1 `attention`", s.ScamsFlagged, s.Scams)
	row("legit with ≥ 1 `attention` (false alerts)", len(s.FalseAlerts), s.Legit)
	row("legit with ≥ 1 `alerte` (must stay 0)", s.LegitAlerted, s.Legit)
	row("cases with a source unavailable", s.Unavailable, s.Scams+s.Legit)
	if len(s.FalseAlerts) > 0 {
		b.WriteString("\nFalse alerts:\n\n")
		for _, f := range s.FalseAlerts {
			fmt.Fprintf(&b, "- `%s` (%s)\n", f.Input, f.Level)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
