// SPDX-License-Identifier: AGPL-3.0-only

package amf

// Columns of a white list row, after the common ones.
const (
	// ColumnAMFNumber is the AMF number of the authorisation or offer
	// ("E2020-001", "D-14-01"), "N/A - Passeport" for a provider passported
	// from another EU country.
	ColumnAMFNumber = "no_amf"
	// ColumnName is the provider or offer name, as the AMF lists it.
	ColumnName = "name"
)
