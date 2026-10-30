// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/arnakoff/api/internal/domain"
)

// options are the command-line flags; see the package doc.
type options struct {
	once   bool
	dryRun bool
	only   domain.Source
}

// errUsage is a misuse of the flags that package flag does not catch.
var errUsage = errors.New("usage")

// parseFlags reads args, writing usage and errors to w. It returns
// flag.ErrHelp for -h.
func parseFlags(args []string, w io.Writer) (options, error) {
	var opts options
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(w)
	fs.BoolVar(&opts.once, "once", false, "sync every source once, then exit")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "with -once: download and validate, store nothing, send no webhook")
	fs.Func("only", "with -once: sync this source only, by its exact name (\"AMF PSAN\")", func(s string) error {
		if s == "" {
			return errors.New("empty source name")
		}
		opts.only = domain.Source(s)
		return nil
	})
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if err := check(opts, fs.NArg()); err != nil {
		_, _ = fmt.Fprintln(w, err)
		fs.Usage()
		return options{}, err
	}
	return opts, nil
}

func check(opts options, extra int) error {
	switch {
	case extra > 0:
		return fmt.Errorf("%w: no argument expected", errUsage)
	case !opts.once && (opts.dryRun || opts.only != ""):
		return fmt.Errorf("%w: -dry-run and -only need -once", errUsage)
	}
	return nil
}
