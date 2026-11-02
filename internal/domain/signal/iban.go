// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/arnakoff/api/internal/domain"
)

// IBANSource names the source of iban_country: the IBAN itself, whose
// country code the ISO 13616 IBAN registry defines.
const IBANSource domain.Source = "IBAN"

// IBANRegistryURL is the public page of the ISO 13616 IBAN registry, kept
// by SWIFT, which lists the country code of every IBAN country.
const IBANRegistryURL = "https://www.swift.com/standards/data-standards/iban-international-bank-account-number"

// IBANInfo describes the source of iban_country for GET /v1/sources:
// nothing is downloaded, the country is read in the IBAN itself.
func IBANInfo() domain.SourceInfo {
	return domain.SourceInfo{
		Name:      IBANSource,
		Publisher: "SWIFT (registre IBAN, norme ISO 13616)",
		Licence:   "Aucune donnée téléchargée",
		URL:       IBANRegistryURL,
	}
}

// frenchRegions names countries in French, from the CLDR data of x/text.
var frenchRegions = display.French.Regions()

// IBANCountry returns the info stating the country holding the account of
// i, as its country code names it, observed at now; false when that code
// has no French name. The signal holds the country only, never the rest of
// the IBAN.
func IBANCountry(i domain.IBAN, now time.Time) (domain.Signal, bool) {
	code := i.Country()
	region, err := language.ParseRegion(code)
	if err != nil {
		return domain.Signal{}, false
	}
	name := frenchRegions.Name(region)
	if name == "" || name == code {
		return domain.Signal{}, false
	}
	return domain.Signal{
		Code:       "iban_country",
		Level:      domain.LevelInfo,
		Message:    "Pays de tenue du compte : " + name + " (code " + code + " de l'IBAN).",
		Source:     IBANSource,
		SourceURL:  IBANRegistryURL,
		ObservedAt: now,
	}, true
}
