// SPDX-License-Identifier: AGPL-3.0-only

package official

import "github.com/arnakoff/api/internal/domain"

// Info describes the list of official domains for GET /v1/sources: facts
// Arnakoff checked itself, kept in this repository (docs/sources.md).
func Info() domain.SourceInfo {
	return domain.SourceInfo{
		Name:      Source,
		Publisher: "Arnakoff",
		Licence:   "Aucune licence tierce",
		URL:       "https://github.com/arnakoff/api/blob/main/internal/adapter/reference/official/domains.csv",
	}
}
