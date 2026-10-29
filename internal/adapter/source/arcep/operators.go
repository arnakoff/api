// SPDX-License-Identifier: AGPL-3.0-only

package arcep

import (
	"fmt"
	"io"
	"strings"

	"github.com/arnakoff/api/internal/domain"
)

// OperatorsURL is the file of the operators ARCEP identifies, with the
// code ("mnémonique") the numbering file names them by.
const OperatorsURL = "https://extranet.arcep.fr/uploads/identifiants_CE.csv"

// Operator is a holder of numbering resources.
type Operator struct {
	// Name is the operator's name, as ARCEP lists it.
	Name string
	// SIREN is the operator's SIREN, "" for a foreign operator or a number
	// that fails its check digit.
	SIREN string
}

// operatorsHeader is the header of the published file (docs/sources.md).
var operatorsHeader = []string{
	"IDENTITE_OPERATEUR", "CODE_OPERATEUR", "SIRET_ACTEUR", "RCS_ACTEUR",
	"ADRESSE_COMPLETE_ACTEUR", "ATTRIB_RESS_NUM", "DATE_DECLARATION_OPERATEUR",
}

// ParseOperators reads the published operators file into a map from code
// to operator. Only the name and the SIREN are kept: the address is not
// needed. A file that departs from the recorded format is rejected with
// ErrFormat rather than half read.
func ParseOperators(r io.Reader) (map[string]Operator, error) {
	cr, err := newReader(r, operatorsHeader)
	if err != nil {
		return nil, err
	}
	ops := make(map[string]Operator)
	err = cr.Each(func(rec []string) error {
		name, code := strings.TrimSpace(rec[0]), strings.TrimSpace(rec[1])
		if name == "" {
			return errEmpty(operatorsHeader[0])
		}
		if code == "" {
			return errEmpty(operatorsHeader[1])
		}
		if _, dup := ops[code]; dup {
			return fmt.Errorf("%w: %s repeated", ErrFormat, operatorsHeader[1])
		}
		ops[code] = Operator{Name: name, SIREN: sirenOf(rec[2])}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ops, nil
}

// sirenOf returns the SIREN of the SIRET_ACTEUR field, which holds a SIRET,
// sometimes a SIREN alone, or for a foreign operator nothing.
func sirenOf(s string) string {
	if n, err := domain.ParseSIRET(s); err == nil {
		return n.SIREN().String()
	}
	if n, err := domain.ParseSIREN(s); err == nil {
		return n.String()
	}
	return ""
}
