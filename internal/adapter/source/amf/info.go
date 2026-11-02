// SPDX-License-Identifier: AGPL-3.0-only

package amf

import "github.com/arnakoff/api/internal/domain"

// Who publishes the AMF lists and under which licence (docs/sources.md):
// the Licence Ouverte 2.0 of their data.gouv.fr datasets, checked on
// 02/11/2026.
const (
	publisher = "Autorité des marchés financiers"
	licence   = "Licence Ouverte 2.0"
)

// BlacklistInfo describes the AMF blacklist for GET /v1/sources.
func BlacklistInfo() domain.SourceInfo {
	return info(BlacklistSource,
		"https://www.data.gouv.fr/datasets/listes-noires-des-entites-non-autorisees-a-proposer-des-produits-ou-services-financiers-en-france")
}

// PSANInfo describes the AMF white list of digital asset service providers
// for GET /v1/sources.
func PSANInfo() domain.SourceInfo {
	return info(PSANSource,
		"https://www.data.gouv.fr/datasets/liste-blanche-des-prestataires-de-services-sur-actifs-numeriques-psan")
}

// OffersInfo describes the AMF white list of offers in goods for
// GET /v1/sources.
func OffersInfo() domain.SourceInfo {
	return info(OffersSource, "https://www.data.gouv.fr/datasets/liste-blanche-des-offres-en-biens-divers")
}

// info describes the AMF list name, whose dataset page is url.
func info(name domain.Source, url string) domain.SourceInfo {
	return domain.SourceInfo{Name: name, Publisher: publisher, Licence: licence, URL: url}
}
