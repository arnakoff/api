// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"

	"github.com/arnakoff/api/internal/domain/signal"
)

// signalCatalog is the body of GET /v1/signals, the contract's
// SignalCatalog.
type signalCatalog struct {
	Signals []signalDefinition `json:"signals"`
}

// signalDefinition is a definition as the contract's SignalDefinition
// describes it.
type signalDefinition struct {
	Code      string   `json:"code"`
	Level     string   `json:"level"`
	Inputs    []string `json:"inputs"`
	Source    string   `json:"source"`
	Condition string   `json:"condition"`
}

// signals answers GET /v1/signals: the catalog of the signals a check can
// give.
func signals() http.Handler {
	body := catalogOf(signal.Catalog())
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, "application/json", http.StatusOK, body)
	})
}

// catalogOf returns the body listing defs.
func catalogOf(defs []signal.Definition) signalCatalog {
	body := signalCatalog{Signals: make([]signalDefinition, 0, len(defs))}
	for _, d := range defs {
		body.Signals = append(body.Signals, signalDefinition{
			Code: d.Code, Level: d.Level.String(), Inputs: d.InputNames(), Source: string(d.Source), Condition: d.Condition,
		})
	}
	return body
}
