// SPDX-License-Identifier: AGPL-3.0-only

// Package bodacc reads the notices the BODACC (DILA, Licence Ouverte;
// docs/sources.md) published about a company, by SIREN only: collective
// procedures, radiations, conciliations and professional recoveries.
//
// The query selects only the notice's family, type, date and references,
// so the names and addresses of the people a notice names are never
// fetched. A cancelled notice is left out, a corrected one gives way to
// its correction. Answers are cached for a day and the API's Retry-After
// is heeded.
package bodacc
