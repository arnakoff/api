// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ErrUnrecognisedInput means the input is not a domain, a web URL, an email
// address, a phone number, an IBAN, a SIREN or a SIRET. When a kind was
// assumed, the error also wraps that kind's error, such as ErrInvalidIBAN.
// The message may quote part of an input read as a domain: log the kind of
// error, matched with errors.Is, not its text.
var ErrUnrecognisedInput = errors.New("unrecognised input")

// Kind is the kind of an Input.
type Kind uint8

// Input kinds. KindAuto, the zero value, is no kind: as a hint to
// ParseInput, it asks for the kind to be detected.
const (
	KindAuto Kind = iota
	KindDomain
	KindEmail
	KindPhone
	KindIBAN
	KindSIREN
	KindSIRET
)

var kindNames = [...]string{"auto", "domain", "email", "phone", "iban", "siren", "siret"}

// String returns the lower-case name of k, such as "iban".
func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "Kind(" + strconv.Itoa(int(k)) + ")"
}

// ErrUnknownKind means a name is not the name of an input kind.
var ErrUnknownKind = errors.New("unknown input kind")

// ParseKind returns the kind named s, as Kind.String writes it, such as
// "iban". "auto" is not a kind and is rejected.
func ParseKind(s string) (Kind, error) {
	for k := KindDomain; int(k) < len(kindNames); k++ {
		if kindNames[k] == s {
			return k, nil
		}
	}
	return KindAuto, fmt.Errorf("%w: %q", ErrUnknownKind, s)
}

// Input is a parsed input: a Domain, an Email, a Phone, an IBAN, a SIREN or
// a SIRET. Use a type switch to reach the value. String returns the
// normalised form, which is personal data for an Email, a Phone and an IBAN.
type Input interface {
	Kind() Kind
	String() string
	isInput()
}

// Kind returns KindDomain.
func (Domain) Kind() Kind { return KindDomain }

// Kind returns KindEmail.
func (Email) Kind() Kind { return KindEmail }

// Kind returns KindPhone.
func (Phone) Kind() Kind { return KindPhone }

// Kind returns KindIBAN.
func (IBAN) Kind() Kind { return KindIBAN }

// Kind returns KindSIREN.
func (SIREN) Kind() Kind { return KindSIREN }

// Kind returns KindSIRET.
func (SIRET) Kind() Kind { return KindSIRET }

func (Domain) isInput() {}
func (Email) isInput()  {}
func (Phone) isInput()  {}
func (IBAN) isInput()   {}
func (SIREN) isInput()  {}
func (SIRET) isInput()  {}

// ParseInput reads what a user pasted into an Input. A hint other than
// KindAuto forces that kind. KindAuto detects the kind from the input's
// shape, in this order:
//
//  1. an "@" outside a URL: an email address ("contact@exemple.fr");
//  2. only letters, digits and spaces, starting with two letters then two
//     digits: an IBAN ("FR14 2004 …");
//  3. only digits and spaces: nine digits are a SIREN, even with a wrong
//     check digit; fourteen digits are a SIRET, or else a phone number
//     ("0044 20 7946 0958"); any other count is a phone number;
//  4. only digits, spaces and "+.-/()": a phone number ("+33 6 12 34 56 78");
//  5. anything else: a domain or a web URL.
//
// The error matches ErrUnrecognisedInput and, once a kind is assumed, the
// error of the first kind tried.
func ParseInput(raw string, hint Kind) (Input, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, ErrUnrecognisedInput
	}
	kinds := []Kind{hint}
	if hint == KindAuto {
		kinds = detectKinds(s)
	}
	var first error
	for _, k := range kinds {
		in, err := parseKind(s, k)
		if err == nil {
			return in, nil
		}
		if first == nil {
			first = err
		}
	}
	return nil, fmt.Errorf("%w: %w", ErrUnrecognisedInput, first)
}

// parseKind parses s as an input of kind k.
func parseKind(s string, k Kind) (Input, error) {
	switch k {
	case KindDomain:
		return ParseDomain(s)
	case KindEmail:
		return ParseEmail(s)
	case KindPhone:
		return ParsePhone(s)
	case KindIBAN:
		return ParseIBAN(s)
	case KindSIREN:
		return ParseSIREN(s)
	case KindSIRET:
		return ParseSIRET(s)
	case KindAuto:
		fallthrough
	default:
		return nil, fmt.Errorf("unknown kind %v", k)
	}
}

// Candidate kinds of each shape, in the order ParseInput tries them. They
// are shared, never modified.
var (
	emailKinds         = []Kind{KindEmail}
	ibanKinds          = []Kind{KindIBAN}
	sirenKinds         = []Kind{KindSIREN}
	fourteenDigitKinds = []Kind{KindSIRET, KindPhone}
	phoneKinds         = []Kind{KindPhone}
	domainKinds        = []Kind{KindDomain}
)

// detectKinds returns the kinds s may be, in the order ParseInput tries
// them.
func detectKinds(s string) []Kind {
	sh := shapeOf(s)
	switch {
	case sh.at && !hasScheme(s):
		return emailKinds
	case sh.alnum && sh.ibanStart:
		return ibanKinds
	case sh.digitsOnly && sh.digits == 9:
		return sirenKinds
	case sh.digitsOnly && sh.digits == 14:
		return fourteenDigitKinds
	case sh.phone && sh.digits > 0:
		return phoneKinds
	default:
		return domainKinds
	}
}

// shape describes the characters of an input.
type shape struct {
	at         bool // holds an "@"
	alnum      bool // only ASCII letters, digits and spaces
	ibanStart  bool // starts with two letters then two digits, spaces aside
	digitsOnly bool // only ASCII digits and spaces
	phone      bool // only ASCII digits, spaces and "+.-/()"
	digits     int  // number of ASCII digits
}

// shapeOf scans s once.
func shapeOf(s string) shape {
	sh := shape{alnum: true, digitsOnly: true, phone: true}
	var isLetter [4]bool // of the first four letters or digits
	n := 0
	for _, r := range s {
		digit := '0' <= r && r <= '9'
		letter := 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z'
		space := unicode.IsSpace(r)
		switch {
		case digit:
			sh.digits++
		case space:
		case letter:
			sh.digitsOnly, sh.phone = false, false
		default:
			sh.alnum, sh.digitsOnly = false, false
			sh.phone = sh.phone && strings.ContainsRune("+.-/()", r)
			sh.at = sh.at || r == '@'
		}
		if (digit || letter) && n < len(isLetter) {
			isLetter[n] = letter
			n++
		}
	}
	sh.ibanStart = n == len(isLetter) && isLetter == [4]bool{true, true, false, false}
	return sh
}
