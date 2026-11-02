// SPDX-License-Identifier: AGPL-3.0-only

package acpr

import (
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/domain/signal"
)

// BlacklistInfo describes the ACPR blacklist for GET /v1/sources. It
// publishes no licence: the source is turned on only once its publisher
// authorises the reuse (issue #30, docs/sources.md).
func BlacklistInfo() domain.SourceInfo {
	return domain.SourceInfo{
		Name:      BlacklistSource,
		Publisher: "ACPR et AMF (ABE Info Service)",
		Licence:   "Autorisation de l'éditeur",
		URL:       signal.ACPRBlacklist().URL(),
	}
}
