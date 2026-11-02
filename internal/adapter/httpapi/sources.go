// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"

	"github.com/arnakoff/api/internal/app"
)

// SourceLister lists the sources a check consults.
type SourceLister interface {
	// Sources returns every source and the state of its data in service.
	Sources() []app.ListedSource
}

// sourceList is the body of GET /v1/sources, the contract's SourceList.
type sourceList struct {
	Sources []sourceBody `json:"sources"`
}

// sourceBody is a source as the contract's Source schema describes it.
// UpdatedAt and Entries are null for a source that is not synced.
type sourceBody struct {
	Name      string  `json:"name"`
	Publisher string  `json:"publisher"`
	Licence   string  `json:"licence"`
	URL       string  `json:"url"`
	Status    string  `json:"status"`
	UpdatedAt *string `json:"updated_at"`
	Entries   *int    `json:"entries"`
}

// Availability of a source, the contract's SourceAvailability.
const (
	sourceOK          = "ok"
	sourceUnavailable = "indisponible"
)

// sources answers GET /v1/sources: every source l lists.
func sources(l SourceLister) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		listed := l.Sources()
		body := sourceList{Sources: make([]sourceBody, 0, len(listed))}
		for _, s := range listed {
			body.Sources = append(body.Sources, sourceOf(s))
		}
		writeJSON(w, "application/json", http.StatusOK, body)
	})
}

// sourceOf returns the body of s. A synced source is available once a
// snapshot is in service; any other is available while it is configured.
func sourceOf(s app.ListedSource) sourceBody {
	body := sourceBody{
		Name: string(s.Info.Name), Publisher: s.Info.Publisher, Licence: s.Info.Licence, URL: s.Info.URL,
		Status: sourceOK,
	}
	if !s.Synced {
		return body
	}
	body.Entries = &s.Sync.Rows
	if s.Sync.LastSuccess.IsZero() {
		body.Status = sourceUnavailable
		return body
	}
	updated := timestamp(s.Sync.LastSuccess)
	body.UpdatedAt = &updated
	return body
}
