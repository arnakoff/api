// SPDX-License-Identifier: AGPL-3.0-only

package wire

import "github.com/arnakoff/api/internal/domain"

// invalidInput is why an input is refused, by the kind it was forced
// into; detected, KindAuto, the input was of no kind and every kind is
// named. A reason never quotes the input, which may be personal data.
var invalidInput = map[domain.Kind]string{
	domain.KindAuto: "L'entrée n'est ni un domaine, ni une URL, ni une adresse e-mail, ni un numéro de téléphone, " +
		"ni un IBAN, ni un SIREN, ni un SIRET.",
	domain.KindDomain: "L'entrée n'est ni un domaine ni une URL valide.",
	domain.KindEmail:  "L'entrée n'est pas une adresse e-mail valide.",
	domain.KindPhone:  "L'entrée n'est pas un numéro de téléphone valide.",
	domain.KindIBAN:   "L'entrée n'est pas un IBAN valide.",
	domain.KindSIREN:  "L'entrée n'est pas un SIREN valide.",
	domain.KindSIRET:  "L'entrée n'est pas un SIRET valide.",
}

// InvalidInput returns why an input of kind k, KindAuto when it was
// detected, is refused.
func InvalidInput(k domain.Kind) string {
	return invalidInput[k]
}
