// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/nyaruka/phonenumbers"
)

// ErrInvalidPhone means the input is not a single dialable phone number. Its
// message never quotes the input, which is personal data.
var ErrInvalidPhone = errors.New("invalid phone number")

// maxPhoneInput bounds the input handed to the parser, in bytes: the longest
// written form of an E.164 number with separators fits well within it.
const maxPhoneInput = 64

// franceCode is the country calling code of metropolitan France.
const franceCode = 33

// overseasCodes maps the first three digits of a nine-digit French national
// significant number to the country calling code of the overseas department
// or region it belongs to. These prefixes are dialled as French numbers
// within France ("0690 12 34 56") but have their own code from abroad
// ("+590 690 12 34 56"). The table lists the geographic and mobile prefixes
// that ARCEP's numbering plan (MAJNUM, "Territoire" column, checked on
// 29/10/2026) assigns to a single overseas code. Prefixes shared by several
// territories, such as 0598, 0939 or 0976, stay French.
var overseasCodes = map[string]int{
	"262": 262, "263": 262, "268": 262, "692": 262, "693": 262, // La Réunion
	"269": 262, "639": 262, // Mayotte
	"590": 590, "690": 590, "691": 590, // Guadeloupe, Saint-Martin, Saint-Barthélemy
	"594": 594, "694": 594, // Guyane
	"596": 596, "696": 596, "697": 596, // Martinique
}

// Phone is a phone number in E.164 form. It is personal data: log Redacted,
// never String. The zero value is not a valid number.
type Phone struct {
	code     int
	national string
}

// ParsePhone reads a phone number as a user would write it, in French
// national form ("06 12 34 56 78") or international form ("+33 6 12 34 56
// 78", "0033 6 12 34 56 78"), with spaces, dots, dashes, slashes or
// brackets between digits. National overseas numbers ("0690 12 34 56") get
// their overseas country code. Letters, extensions and short numbers are
// rejected. The number is checked for a plausible length, not for being in
// service: that is ARCEP's numbering plan's job.
func ParsePhone(s string) (Phone, error) {
	in, ok := cleanPhoneInput(s)
	if !ok {
		return Phone{}, ErrInvalidPhone
	}
	p, ok := readPhone(in)
	if !ok {
		return Phone{}, ErrInvalidPhone
	}
	// Some inputs, such as a national number starting with a trunk prefix
	// after a country code, read differently once in E.164 form: reject
	// them, so that every Phone reads back as itself.
	if again, ok := readPhone(p.String()); !ok || again != p {
		return Phone{}, ErrInvalidPhone
	}
	return p, nil
}

// readPhone parses a cleaned input with France as the default region and
// gives a French national overseas number its overseas country code.
func readPhone(in string) (Phone, bool) {
	n, err := phonenumbers.Parse(in, "FR")
	if err != nil || !phonenumbers.IsPossibleNumber(n) {
		// The cause quotes the input: drop it.
		return Phone{}, false
	}
	p := Phone{code: int(n.GetCountryCode()), national: phonenumbers.GetNationalSignificantNumber(n)}
	if p.code == franceCode && len(p.national) == 9 {
		if code, ok := overseasCodes[p.national[:3]]; ok {
			p.code = code
		}
	}
	return p, true
}

// cleanPhoneInput trims s and turns every space, including no-break spaces,
// into an ASCII space. It reports false for an input too long or holding
// anything but digits, spaces and the separators "+.-/()".
func cleanPhoneInput(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > maxPhoneInput {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', strings.ContainsRune("+.-/()", r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		default:
			return "", false
		}
	}
	return b.String(), true
}

// String returns the number in E.164 form, such as "+33612345678".
func (p Phone) String() string {
	return "+" + strconv.Itoa(p.code) + p.national
}

// CountryCode returns the country calling code, such as 33.
func (p Phone) CountryCode() int {
	return p.code
}

// FrenchNational returns the number as dialled within France, ten digits
// starting with 0 ("0612345678"), the form ARCEP's numbering plan ranges
// use. It reports false for a number outside the French numbering plan
// (metropolitan France and the overseas departments and regions above).
func (p Phone) FrenchNational() (string, bool) {
	if len(p.national) != 9 || !inFrenchPlan(p.code) {
		return "", false
	}
	return "0" + p.national, true
}

// inFrenchPlan reports whether code is metropolitan France's or one of the
// overseas codes of overseasCodes.
func inFrenchPlan(code int) bool {
	switch code {
	case franceCode, 262, 590, 594, 596:
		return true
	default:
		return false
	}
}

// Redacted returns a form safe for logs, the country code and the last two
// digits only: "+33***78". The zero value gives "***".
func (p Phone) Redacted() string {
	if p.national == "" {
		return "***"
	}
	return "+" + strconv.Itoa(p.code) + "***" + p.national[len(p.national)-2:]
}
