// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arnakoff/api/internal/adapter/signaldoc"
	"github.com/arnakoff/api/internal/domain/signal"
)

func main() {
	file := flag.String("file", "docs/signals.md", "the document to rewrite")
	flag.Parse()
	if err := run(*file); err != nil {
		fmt.Fprintln(os.Stderr, "signaldoc:", err)
		os.Exit(1)
	}
}

// run rewrites the table of the document at file from the catalog,
// keeping its mode; a document without the markers is left unchanged.
func run(file string) (err error) {
	root, err := os.OpenRoot(filepath.Dir(file))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	name := filepath.Base(file)
	info, err := root.Stat(name)
	if err != nil {
		return err
	}
	doc, err := root.ReadFile(name)
	if err != nil {
		return err
	}
	out, err := signaldoc.Rewrite(string(doc), signaldoc.Table(signal.Catalog()))
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	return root.WriteFile(name, []byte(out), info.Mode().Perm())
}
