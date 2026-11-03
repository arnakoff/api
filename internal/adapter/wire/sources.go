// SPDX-License-Identifier: AGPL-3.0-only

package wire

import "github.com/arnakoff/api/internal/app"

// SourceList is the list of sources, the contract's SourceList.
type SourceList struct {
	Sources []Source `json:"sources"`
}

// Source is a source as the contract's Source schema describes it.
// UpdatedAt and Entries are null for a source that is not synced.
type Source struct {
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

// SourcesOf returns the list of the sources listed, in their order.
func SourcesOf(listed []app.ListedSource) SourceList {
	list := SourceList{Sources: make([]Source, 0, len(listed))}
	for _, s := range listed {
		list.Sources = append(list.Sources, sourceOf(s))
	}
	return list
}

// sourceOf returns the representation of s. A synced source is available
// once a snapshot is in service; any other is available while it is
// configured.
func sourceOf(s app.ListedSource) Source {
	body := Source{
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
