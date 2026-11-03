// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"

	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/app"
)

// SourceLister lists the sources a check consults.
type SourceLister interface {
	// Sources returns every source and the state of its data in service.
	Sources() []app.ListedSource
}

// sources answers GET /v1/sources: every source l lists.
func sources(l SourceLister) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, "application/json", http.StatusOK, wire.SourcesOf(l.Sources()))
	})
}
