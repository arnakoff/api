// SPDX-License-Identifier: AGPL-3.0-only

package acpr_test

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"slices"
	"testing"

	"github.com/arnakoff/api/internal/adapter/source/acpr"
	"github.com/arnakoff/api/internal/adapter/source/amf"
	"github.com/arnakoff/api/internal/domain"
)

// testdata holds the trimmed real files of every source.
var testdata = os.DirFS("../../../../testdata")

// keys returns the kind, key and path of every row of the fixture file
// parsed by parse.
func keys(t *testing.T, file string, parse func(io.Reader) (domain.Snapshot, error)) map[[3]string]bool {
	t.Helper()
	b, err := fs.ReadFile(testdata, file)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	snap, err := parse(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	out := make(map[[3]string]bool, len(snap.Rows))
	for _, r := range snap.Rows {
		out[[3]string{r[0], r[1], r[2]}] = true
	}
	return out
}

// TestKeysMatchTheAMFBlacklist checks that an entry both blacklists hold,
// even written differently ("http://audubert-conseil.com" and
// "audubert-conseil.com"), gets the same key in both snapshots: a check can
// then report it once, naming both sources.
func TestKeysMatchTheAMFBlacklist(t *testing.T) {
	t.Parallel()
	amfKeys := keys(t, "amf/listesnoires.csv", amf.ParseBlacklist)
	acprKeys := keys(t, "acpr/liste-noire.csv", acpr.ParseBlacklist)
	var common []string
	for k := range acprKeys {
		if amfKeys[k] {
			common = append(common, k[1]+k[2])
		}
	}
	slices.Sort(common)
	want := []string{
		"alphaairobot.com", "armen.ltd", "audubert-conseil.com", "calendly.com/amf-france",
		"hlf-gestion-or.com", "nogentais.finance", "optimalduck.autos", "prénom.nom@paris-majeste.com", "support@dbf-link.com",
	}
	if !slices.Equal(common, want) {
		t.Errorf("keys in both blacklists = %q, want %q", common, want)
	}
}
