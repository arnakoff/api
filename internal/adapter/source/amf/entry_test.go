// SPDX-License-Identifier: AGPL-3.0-only

package amf

import "testing"

func TestClassify(t *testing.T) {
	t.Parallel()
	tests := []struct {
		nom             string
		kind, key, path string
		ok              bool
	}{
		{nom: "armen.ltd", kind: "domain", key: "armen.ltd", ok: true},
		{nom: "www.BourseBinaire.fr", kind: "domain", key: "www.boursebinaire.fr", ok: true},
		{nom: "https://www.groupe-ytria.com", kind: "domain", key: "www.groupe-ytria.com", ok: true},
		{nom: "http://audubert-conseil.com", kind: "domain", key: "audubert-conseil.com", ok: true},
		{nom: "ttps://www.alphacap.fr/", kind: "domain", key: "www.alphacap.fr", ok: true},
		{nom: "ftp://armen.ltd"},
		{nom: "optimalduck.autos/?", kind: "domain", key: "optimalduck.autos", ok: true},
		{nom: "calendly.com/amf-france", kind: "domain", key: "calendly.com", path: "/amf-france", ok: true},
		{nom: "www.vantagemarkets.com/fr/", kind: "domain", key: "www.vantagemarkets.com", path: "/fr/", ok: true},
		{nom: "repubblicait5.com/?_lp", kind: "domain", key: "repubblicait5.com", path: "/?_lp", ok: true},
		{nom: "support@Dbf-Link.com", kind: "email", key: "support@dbf-link.com", ok: true},
		{nom: "prénom.nom@paris-majeste.com", kind: "email", key: "prénom.nom@paris-majeste.com", ok: true},
		{nom: "prénom.nom@lgt-bp.com.", kind: "email", key: "prénom.nom@lgt-bp.com", ok: true},
		{nom: "support@amf-france.tram"},
		{nom: " armen.ltd ", kind: "domain", key: "armen.ltd", ok: true},
		{nom: "Kuvera LLC / Kuvera France"},
		{nom: "www.BourseBinaire.fr (Plan B)"},
		{nom: "CapitalFrance"},
		{nom: ""},
	}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			t.Parallel()
			k, ok := classify(tt.nom)
			if k.kind != tt.kind || k.value != tt.key || k.path != tt.path || ok != tt.ok {
				t.Errorf("classify(%q) = %+v, %v, want %q, %q, %q, %v",
					tt.nom, k, ok, tt.kind, tt.key, tt.path, tt.ok)
			}
		})
	}
}
