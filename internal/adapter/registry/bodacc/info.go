// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import "github.com/arnakoff/api/internal/domain"

// Info describes the BODACC for GET /v1/sources: the Licence Ouverte 1.0
// (fr-lo) of its data.gouv.fr dataset, checked on 02/11/2026.
func Info() domain.SourceInfo {
	return domain.SourceInfo{
		Name:      Source,
		Publisher: "DILA",
		Licence:   "Licence Ouverte 1.0",
		URL:       "https://www.data.gouv.fr/datasets/bodacc",
	}
}
