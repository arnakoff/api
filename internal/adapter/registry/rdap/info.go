// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import "github.com/arnakoff/api/internal/domain"

// Info describes RDAP for GET /v1/sources: the registry of each extension,
// found through the IANA bootstrap, whose page lists them. Only the
// registration events of a domain are read (docs/sources.md).
func Info() domain.SourceInfo {
	return domain.SourceInfo{
		Name:      Source,
		Publisher: "Registres des extensions (amorçage IANA)",
		Licence:   "Données publiques d'enregistrement",
		URL:       "https://www.iana.org/assignments/rdap-dns",
	}
}
