// SPDX-License-Identifier: AGPL-3.0-only

// Package logging builds the structured logger every command writes with:
// log/slog lines in JSON, personal data masked, the request identifier of
// the context added (ADR 0002).
//
// A value with both a Redacted and a String method, such as domain.Email,
// domain.Phone or domain.IBAN, is never written as such: the logger writes
// an object holding its Redacted form and a keyed pseudonym of its String
// form, {"redacted":"***@exemple.fr","hmac":"…"}. The pseudonym is the
// first 16 bytes of HMAC-SHA256 under a secret key, in hex: the same value
// gets the same pseudonym, so lines about it can be grouped, but it cannot
// be recovered nor guessed by hashing candidates without the key.
//
// Masking relies on the value reaching the logger whole: logging
// email.String() writes the address. Log the value object itself.
package logging
