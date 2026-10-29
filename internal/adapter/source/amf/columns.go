// SPDX-License-Identifier: AGPL-3.0-only

package amf

// Columns every list's snapshot rows start with.
const (
	// ColumnKind is the kind of the key: "domain", "email", and for the
	// white lists "phone" or "siren".
	ColumnKind = "kind"
	// ColumnKey is the normalised host, address, number or SIREN the entry
	// is looked up by.
	ColumnKey = "key"
	// ColumnPath is the path and query of a site entry that targets one
	// page, "" when it targets the whole site or the key is not a site.
	ColumnPath = "path"
	// ColumnEntry is the value as the AMF lists it.
	ColumnEntry = "entry"
)

// Columns of a white list row, after the common ones.
const (
	// ColumnAMFNumber is the AMF number of the authorisation or offer
	// ("E2020-001", "D-14-01"), "N/A - Passeport" for a provider passported
	// from another EU country.
	ColumnAMFNumber = "no_amf"
	// ColumnName is the provider or offer name, as the AMF lists it.
	ColumnName = "name"
)
