// SPDX-License-Identifier: AGPL-3.0-only

package wire_test

import (
	"testing"

	"github.com/arnakoff/api/internal/adapter/wire"
	"github.com/arnakoff/api/internal/domain"
)

func TestInvalidInputNamesTheKindTheInputWasForcedInto(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		kind domain.Kind
		want string
	}{
		{domain.KindAuto, "L'entrée n'est ni un domaine, ni une URL, ni une adresse e-mail, ni un numéro de téléphone, " +
			"ni un IBAN, ni un SIREN, ni un SIRET."},
		{domain.KindDomain, "L'entrée n'est ni un domaine ni une URL valide."},
		{domain.KindEmail, "L'entrée n'est pas une adresse e-mail valide."},
		{domain.KindPhone, "L'entrée n'est pas un numéro de téléphone valide."},
		{domain.KindIBAN, "L'entrée n'est pas un IBAN valide."},
		{domain.KindSIREN, "L'entrée n'est pas un SIREN valide."},
		{domain.KindSIRET, "L'entrée n'est pas un SIRET valide."},
	} {
		if got := wire.InvalidInput(tt.kind); got != tt.want {
			t.Errorf("InvalidInput(%v) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}
