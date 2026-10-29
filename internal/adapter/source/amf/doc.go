// SPDX-License-Identifier: AGPL-3.0-only

// Package amf holds the connectors of the lists the Autorité des marchés
// financiers publishes daily on data.gouv.fr under the Licence Ouverte 2.0
// (docs/sources.md): the blacklist, and the white lists of digital asset
// service providers (PSAN) and of offers in goods (biens divers). Each list
// is its own source, with its own snapshot.
//
// Each snapshot row is one key of an entry, looked up by its normalised
// host, address, phone number or SIREN. Free-text names of the blacklist,
// which may name natural persons, are left out: no input kind matches them.
package amf
