// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

// ErrInvalidSIRET means the input is not a valid SIRET number.
var ErrInvalidSIRET = errors.New("invalid SIRET")

// laPosteSIREN is the SIREN of La Poste, whose establishments do not follow
// the Luhn check.
const laPosteSIREN = "356000000"

// SIRET is the fourteen-digit number the SIRENE register gives to an
// establishment: the company's SIREN followed by a five-digit NIC. The zero
// value is not a valid SIRET.
type SIRET struct {
	value string
}

// ParseSIRET reads a SIRET as a user would write it, fourteen digits alone
// ("12002701600563") or in groups ("120 027 016 00563") with any spaces
// between them. The SIREN part and the whole number are checked; the
// establishment is not checked for existing.
func ParseSIRET(s string) (SIRET, error) {
	in, ok := cleanSIRENInput(s)
	if !ok || len(in) != 14 || !luhnValid(in[:9]) || !siretKeyValid(in) {
		return SIRET{}, ErrInvalidSIRET
	}
	return SIRET{value: in}, nil
}

// siretKeyValid reports whether the fourteen digits of s pass the check of a
// SIRET. Most pass the Luhn check. The SIRETs of La Poste establishments in
// the SIRENE register do not, such as 35600000009075 and 35600000067814, but
// their digits add up to a multiple of 5, while its head office,
// 35600000000048, passes the Luhn check: both are accepted for La Poste. INSEE
// does not document this exception (https://xml.insee.fr/schema/siret.html,
// section 8.1, « à vérifier »): the rule is read from the register, and is the
// one python-stdnum applies (stdnum/fr/siret.py).
func siretKeyValid(s string) bool {
	if luhnValid(s) {
		return true
	}
	return s[:9] == laPosteSIREN && digitSum(s)%5 == 0
}

// digitSum adds up the ASCII digits of s.
func digitSum(s string) int {
	sum := 0
	for i := range len(s) {
		sum += int(s[i] - '0')
	}
	return sum
}

// String returns the fourteen digits, such as "12002701600563".
func (n SIRET) String() string {
	return n.value
}

// SIREN returns the SIREN of the company the establishment belongs to, the
// first nine digits, such as "120027016".
func (n SIRET) SIREN() SIREN {
	return SIREN{value: n.value[:9]}
}

// NIC returns the five-digit establishment number ("numéro interne de
// classement"), the last five digits, such as "00563".
func (n SIRET) NIC() string {
	return n.value[9:]
}
