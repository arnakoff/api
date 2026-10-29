// SPDX-License-Identifier: AGPL-3.0-only

package api_test

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

const spdxHeader = "// SPDX-License-Identifier: AGPL-3.0-only"

// skippedDirs are never scanned: they hold no source of this module.
var skippedDirs = map[string]bool{".git": true, ".claude": true, "var": true, "dist": true, "testdata": true}

func TestGoFiles_startWithSPDXHeader(t *testing.T) {
	t.Parallel()

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && skippedDirs[d.Name()] {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		if line := firstLine(t, path); line != spdxHeader {
			t.Errorf("%s: first line is %q, want %q", path, line, spdxHeader)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk module: %v", err)
	}
}

func firstLine(t *testing.T, path string) string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return ""
	}
	return sc.Text()
}
