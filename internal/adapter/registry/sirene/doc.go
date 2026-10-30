// SPDX-License-Identifier: AGPL-3.0-only

// Package sirene reads a company's name, creation date and head office
// state from the API Recherche d'entreprises (DINUM, INSEE data, Licence
// Ouverte 2.0; docs/sources.md), by SIREN only.
//
// The query asks for the minimal answer with the head office, so the
// dirigeants, who are people, are never fetched. Requests keep to the
// API's published limit of 7 per second and heed its Retry-After. Answers
// are cached for a day, "not found" for a few minutes.
package sirene
