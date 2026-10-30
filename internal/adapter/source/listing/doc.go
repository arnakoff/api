// SPDX-License-Identifier: AGPL-3.0-only

// Package listing turns the entries of the published lists into the keys
// their snapshot rows are looked up by: a site's normalised host and path,
// an address, an E.164 phone number or a SIREN, normalised by the domain
// value objects. Lists that share entries, such as the AMF and ACPR
// blacklists, give them the same key, so a check can name both sources.
package listing
