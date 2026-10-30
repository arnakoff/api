// SPDX-License-Identifier: AGPL-3.0-only

package listing

import (
	"slices"
	"testing"
)

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
			k, ok := Classify(tt.nom)
			if k.Kind != tt.kind || k.Value != tt.key || k.Path != tt.path || ok != tt.ok {
				t.Errorf("Classify(%q) = %+v, %v, want %q, %q, %q, %v",
					tt.nom, k, ok, tt.kind, tt.key, tt.path, tt.ok)
			}
		})
	}
}

func TestColumnsAndFields(t *testing.T) {
	t.Parallel()
	cols := Columns(ColumnCategories, ColumnListedOn)
	if want := []string{"kind", "key", "path", "entry", "categories", "listed_on"}; !slices.Equal(cols, want) {
		t.Errorf("Columns() = %q, want %q", cols, want)
	}
	k, _ := Classify("calendly.com/amf-france")
	if got, want := k.Fields("Forex", "2024-01-09"), []string{"domain", "calendly.com", "/amf-france", "calendly.com/amf-france", "Forex", "2024-01-09"}; !slices.Equal(got, want) {
		t.Errorf("Fields() = %q, want %q", got, want)
	}
}

func TestPhoneAndSIREN(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		read     func(string) (Key, bool)
		in, want string
		ok       bool
	}{
		{"phone", Phone, "+33 (0)5 57 93 41 59", "+33557934159", true},
		{"phone with a final space", Phone, "+33 1 82 88 29 51 ", "+33182882951", true},
		{"two phones", Phone, "+33 1 82 88 29 51 / +33 1 76 36 05 18", "", false},
		{"siren", SIREN, "815254545", "815254545", true},
		{"passport number", SIREN, "N/A - Passeport", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k, ok := tt.read(tt.in)
			if k.Value != tt.want || ok != tt.ok {
				t.Errorf("read(%q) = %+v, %v, want %q, %v", tt.in, k, ok, tt.want, tt.ok)
			}
		})
	}
}
