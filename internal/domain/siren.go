// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"strings"
	"unicode"
)

// ErrInvalidSIREN means the input is not a valid SIREN number.
var ErrInvalidSIREN = errors.New("invalid SIREN")

// maxSIRENInput bounds the input handed to the SIREN and SIRET parsers, in
// bytes: a SIRET in groups with three-byte spaces between them fits well
// within it.
const maxSIRENInput = 32

// SIREN is the nine-digit number the SIRENE register gives to a company. The
// zero value is not a valid SIREN.
type SIREN struct {
	value string
}

// ParseSIREN reads a SIREN as a user would write it, nine digits alone
// ("120027016") or in groups ("120 027 016") with any spaces between them.
// The Luhn check digit is validated; the company is not checked for existing.
func ParseSIREN(s string) (SIREN, error) {
	in, ok := cleanSIRENInput(s)
	if !ok || len(in) != 9 || !luhnValid(in) {
		return SIREN{}, ErrInvalidSIREN
	}
	return SIREN{value: in}, nil
}

// String returns the nine digits, such as "120027016".
func (n SIREN) String() string {
	return n.value
}

// cleanSIRENInput drops every space, including no-break spaces. It reports
// false for an input too long or holding anything but ASCII digits and
// spaces.
func cleanSIRENInput(s string) (string, bool) {
	if len(s) > maxSIRENInput {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case unicode.IsSpace(r):
		default:
			return "", false
		}
	}
	return b.String(), true
}

// luhnValid reports whether the ASCII digits of s pass the Luhn check, the
// key of SIREN and SIRET numbers described by INSEE
// (https://xml.insee.fr/schema/siret.html, section 8.1): every second digit
// from the right is doubled, the digits of the results are added up and the
// total is a multiple of 10.
func luhnValid(s string) bool {
	sum := 0
	for i := range len(s) {
		d := int(s[len(s)-1-i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}
