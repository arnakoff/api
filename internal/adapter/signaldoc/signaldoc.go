// SPDX-License-Identifier: AGPL-3.0-only

// Package signaldoc writes the table of docs/signals.md from the signal
// catalog, so that the public list cannot drift from the rules.
package signaldoc

//go:generate go run ../../../cmd/signaldoc -file ../../../docs/signals.md

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arnakoff/api/internal/domain/signal"
)

// Markers around the generated table in docs/signals.md.
const (
	Begin = "<!-- Début du tableau généré depuis le catalogue (go generate ./...) : ne pas modifier à la main. -->"
	End   = "<!-- Fin du tableau généré. -->"
)

// ErrMarkers means a document does not hold Begin then End.
var ErrMarkers = errors.New("signal table markers missing")

// Table returns the Markdown table of defs, one row each, in order.
func Table(defs []signal.Definition) string {
	var t strings.Builder
	t.WriteString("| Code | Entrées | Niveau | Source | Condition |\n|---|---|---|---|---|\n")
	for _, d := range defs {
		fmt.Fprintf(&t, "| `%s` | %s | %s | %s | %s |\n",
			d.Code, strings.Join(d.InputNames(), ", "), d.Level, cell(string(d.Source)), cell(d.Condition))
	}
	return t.String()
}

// cell escapes the pipes of s, which would end a table cell.
func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

// Rewrite returns doc with table between its markers, each on its own
// line; ErrMarkers when doc does not hold Begin then End.
func Rewrite(doc, table string) (string, error) {
	before, rest, found := strings.Cut(doc, Begin+"\n")
	if !found {
		return "", fmt.Errorf("%w: no %q line", ErrMarkers, Begin)
	}
	_, after, found := strings.Cut(rest, End)
	if !found {
		return "", fmt.Errorf("%w: no %q after the beginning", ErrMarkers, End)
	}
	return before + Begin + "\n" + table + End + after, nil
}
