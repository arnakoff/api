// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/signaldoc"
)

func TestRunWritesTheCatalogTable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "signals.md")
	doc := "# Signaux\n\n" + signaldoc.Begin + "\nold\n" + signaldoc.End + "\n\n## Suite\n"
	if err := os.WriteFile(file, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run(file); err != nil {
		t.Fatalf("run: %v", err)
	}

	got, err := fs.ReadFile(os.DirFS(dir), "signals.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "| `amf_blacklist` | domain, email | alerte | AMF |") ||
		!strings.HasSuffix(string(got), signaldoc.End+"\n\n## Suite\n") {
		t.Errorf("signals.md =\n%s\nwant the catalog table between the markers", got)
	}
}

func TestRunLeavesAFileWithoutMarkers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "signals.md")
	if err := os.WriteFile(file, []byte("# Signaux\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run(file); !errors.Is(err, signaldoc.ErrMarkers) {
		t.Errorf("run error = %v, want ErrMarkers", err)
	}
	if got, _ := fs.ReadFile(os.DirFS(dir), "signals.md"); string(got) != "# Signaux\n" {
		t.Errorf("signals.md = %q, want it unchanged", got)
	}
}
