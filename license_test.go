// SPDX-License-Identifier: AGPL-3.0-only

package api_test

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"testing"
)

const spdxHeader = "// SPDX-License-Identifier: AGPL-3.0-only"

// skippedDirs are never scanned: they hold no source of this module.
var skippedDirs = map[string]bool{".git": true, ".claude": true, "var": true, "dist": true, "testdata": true}

func TestGoFiles_startWithSPDXHeader(t *testing.T) {
	t.Parallel()

	module := os.DirFS(".")
	err := fs.WalkDir(module, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && skippedDirs[d.Name()] {
			return fs.SkipDir
		}
		if d.IsDir() || path.Ext(name) != ".go" {
			return nil
		}
		if line := firstLine(t, module, name); line != spdxHeader {
			t.Errorf("%s: first line is %q, want %q", name, line, spdxHeader)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk module: %v", err)
	}
}

func firstLine(t *testing.T, fsys fs.FS, name string) string {
	t.Helper()

	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	return string(bytes.TrimSuffix(line, []byte("\r")))
}
