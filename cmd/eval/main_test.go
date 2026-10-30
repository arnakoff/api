// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const corpusHeader = "input,type,label,added_at,note\n"

func TestRunOfflineScoresTheCorpus(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "scam.csv", corpusHeader+"armen.ltd,domain,scam,2026-10-08,liste noire AMF\n")
	writeFile(t, dir, "legit.csv", corpusHeader+"www.impots.gouv.fr,domain,legit,2026-10-30,service-public\n")
	if err := os.Mkdir(filepath.Join(dir, "replay"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "replay"), "rdap.json",
		`{"impots.gouv.fr": {"value": {"registered": "1999-01-01T00:00:00Z"}}}`)
	var out strings.Builder

	err := run(context.Background(), options{corpus: dir, offline: true, without: []string{"amf", "acpr"}}, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Sources left out: amf, acpr.",
		"| scams with ≥ 1 `alerte` | 0 / 1 |",
		// armen.ltd was not recorded: its RDAP source is unavailable.
		"| cases with a source unavailable | 1 / 2 |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report lacks %q:\n%s", want, got)
		}
	}
}

func TestRunOfflineLeavesOutTheRegistries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "scam.csv", corpusHeader+"armen.ltd,domain,scam,2026-10-08,\n")
	writeFile(t, dir, "legit.csv", corpusHeader+"120027016,siren,legit,2026-10-30,\n")
	var out strings.Builder

	err := run(context.Background(), options{corpus: dir, offline: true, without: []string{"rdap", "sirene", "bodacc"}}, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if want := "| cases with a source unavailable | 0 / 2 |"; !strings.Contains(out.String(), want) {
		t.Errorf("report lacks %q:\n%s", want, out.String())
	}
}

func TestRunRejectsAMissingCorpus(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), options{corpus: filepath.Join(t.TempDir(), "missing"), offline: true}, &strings.Builder{})
	if err == nil {
		t.Error("run of a missing corpus: want an error")
	}
}

// The live registries are built without reaching the network.
func TestLiveRegistries(t *testing.T) {
	t.Parallel()

	r, err := liveRegistries()
	if err != nil {
		t.Fatalf("liveRegistries: %v", err)
	}
	if r.Domains.Source() != "RDAP" || r.Companies.Source() != "SIRENE" || r.Notices.Source() != "BODACC" {
		t.Errorf("liveRegistries = %+v, want RDAP, SIRENE and BODACC", r)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
