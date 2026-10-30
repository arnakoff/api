// SPDX-License-Identifier: AGPL-3.0-only

// Package acpr holds the connector of the blacklist the Autorité de
// contrôle prudentiel et de résolution and the AMF publish through
// Assurance Banque Épargne Info Service (docs/sources.md). Its reuse
// licence is still to be obtained (issue #30): the source stays disabled
// by default until then.
//
// Each snapshot row is one site or address, keyed as the AMF blacklist
// keys it, so an entry both lists hold can be reported with both sources.
// Free-text names and the Dénomination column, which may name natural
// persons, are left out.
package acpr
