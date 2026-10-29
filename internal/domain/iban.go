// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"strings"
	"unicode"

	"github.com/jacoelho/banking/iban"
)

// ErrInvalidIBAN means the input is not a valid IBAN. Its message never
// quotes the input, which is personal data.
var ErrInvalidIBAN = errors.New("invalid IBAN")

// maxIBANInput bounds the input handed to the parser, in bytes: the paper
// form of the longest IBAN, 34 characters in groups of four, fits well
// within it.
const maxIBANInput = 64

// IBAN is an International Bank Account Number in electronic form, upper case
// without spaces. It is personal data: log Redacted, never String. The zero
// value is not a valid IBAN.
type IBAN struct {
	value string
}

// ParseIBAN reads an IBAN as a user would write it, in electronic form
// ("FR1420041010050500013M02606") or paper form ("FR14 2004 1010 0505 0001
// 3M02 606"), in any case and with any spaces between characters. The
// country's length and format and the check digits are validated; the
// account is not checked for existing.
func ParseIBAN(s string) (IBAN, error) {
	in, ok := cleanIBANInput(s)
	if !ok {
		return IBAN{}, ErrInvalidIBAN
	}
	if err := iban.Validate(in); err != nil {
		// The cause quotes the input: drop it.
		return IBAN{}, ErrInvalidIBAN
	}
	return IBAN{value: in}, nil
}

// cleanIBANInput drops every space, including no-break spaces, and upper
// cases ASCII letters. It reports false for an input too long, empty once
// cleaned or holding anything but ASCII letters, digits and spaces.
func cleanIBANInput(s string) (string, bool) {
	if len(s) > maxIBANInput {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
		case unicode.IsSpace(r):
		default:
			return "", false
		}
	}
	return b.String(), b.Len() > 0
}

// String returns the IBAN in electronic form, such as
// "FR1420041010050500013M02606".
func (i IBAN) String() string {
	return i.value
}

// Country returns the ISO 3166-1 alpha-2 code of the country holding the
// account, the IBAN's first two letters, such as "FR".
func (i IBAN) Country() string {
	return i.value[:2]
}

// FrenchBankCode returns the five-digit bank code ("code banque") of a
// French IBAN, the first five characters after the check digits, such as
// "20041". It reports false for an IBAN of any other country.
func (i IBAN) FrenchBankCode() (string, bool) {
	if i.Country() != "FR" {
		return "", false
	}
	return i.value[4:9], true
}

// Redacted returns a form safe for logs, the country and the last four
// characters only: "FR***2606".
func (i IBAN) Redacted() string {
	return i.value[:2] + "***" + i.value[len(i.value)-4:]
}
