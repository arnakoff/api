// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// NumberingInfo describes the ARCEP numbering plan for GET /v1/sources:
// the Licence Ouverte 1.0 (fr-lo) of its data.gouv.fr dataset, checked on
// 02/11/2026.
func NumberingInfo() domain.SourceInfo {
	return domain.SourceInfo{
		Name: NumberingSource,
		Publisher: "Autorité de régulation des communications électroniques, des postes et " +
			"de la distribution de la presse (ARCEP)",
		Licence: "Licence Ouverte 1.0",
		URL:     signal.NumberingURL,
	}
}
