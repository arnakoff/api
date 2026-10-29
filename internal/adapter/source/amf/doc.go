// SPDX-License-Identifier: AGPL-3.0-only

// Package amf is the connector of the AMF blacklist: the CSV file the
// Autorité des marchés financiers publishes daily on data.gouv.fr under the
// Licence Ouverte 2.0 (docs/sources.md).
//
// Each snapshot row is one site or address, looked up by its normalised
// host or address. Free-text names, which may name natural persons, are
// left out: no input kind matches them.
package amf
