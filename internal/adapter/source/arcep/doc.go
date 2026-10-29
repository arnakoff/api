// SPDX-License-Identifier: AGPL-3.0-only

// Package arcep holds the connector of the national numbering plan the
// Autorité de régulation des communications électroniques, des postes et de
// la distribution de la presse publishes under the Licence Ouverte
// (docs/sources.md): the ranges of phone numbers it has assigned (MAJNUM),
// joined with the name and SIREN of the operator each range is assigned to
// (identifiants_CE). The files are Windows-1252 ("ANSI") and replaced
// whole after each update.
//
// Each snapshot row is one assigned range. Index looks a phone number up
// among them. An assignment says which operator holds a number's range,
// nothing about who uses the number.
package arcep
