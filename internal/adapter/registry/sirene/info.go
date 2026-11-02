// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import "github.com/arnakoff/api/internal/domain"

// Info describes SIRENE for GET /v1/sources: the API Recherche
// d'entreprises of the DINUM, serving INSEE data under the Licence
// Ouverte 2.0 (docs/sources.md); its page checked on 02/11/2026.
func Info() domain.SourceInfo {
	return domain.SourceInfo{
		Name:      Source,
		Publisher: "DINUM, données INSEE",
		Licence:   "Licence Ouverte 2.0",
		URL:       "https://www.data.gouv.fr/dataservices/api-recherche-dentreprises",
	}
}
