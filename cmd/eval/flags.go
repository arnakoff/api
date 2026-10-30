// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/arnakoff/api/internal/adapter/registry/bodacc"
	"github.com/arnakoff/api/internal/adapter/registry/rdap"
	"github.com/arnakoff/api/internal/adapter/registry/sirene"
	"github.com/arnakoff/api/internal/adapter/source/acpr"
	"github.com/arnakoff/api/internal/adapter/source/amf"
	"github.com/arnakoff/api/internal/domain"
)

// defaultCorpus is the corpus directory, relative to the repository root.
const defaultCorpus = "testdata/corpus"

// leavable are the sources -without may name, lower case. The blacklists
// are named already: no rule reads them yet (#41).
var leavable = lowerNames(amf.BlacklistSource, acpr.BlacklistSource, rdap.Source, sirene.Source, bodacc.Source)

// lowerNames returns the lower-case names of sources.
func lowerNames(sources ...domain.Source) []string {
	names := make([]string, len(sources))
	for i, s := range sources {
		names[i] = strings.ToLower(string(s))
	}
	return names
}

// options are the command-line flags; see the package doc.
type options struct {
	corpus  string
	offline bool
	without []string // lower case
}

// leftOut reports whether the run leaves s out.
func (o options) leftOut(s domain.Source) bool {
	return slices.Contains(o.without, strings.ToLower(string(s)))
}

// errUsage is a misuse of the flags that package flag does not catch.
var errUsage = errors.New("usage")

// parseFlags reads args, writing usage and errors to w. It returns
// flag.ErrHelp for -h.
func parseFlags(args []string, w io.Writer) (options, error) {
	opts := options{corpus: defaultCorpus}
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(w)
	fs.StringVar(&opts.corpus, "corpus", defaultCorpus, "corpus directory, holding scam.csv, legit.csv and replay/")
	fs.BoolVar(&opts.offline, "offline", false, "replay the recorded registry answers instead of asking the registries")
	fs.Func("without", "comma-separated sources to leave out: "+strings.Join(leavable, ", "), func(s string) error {
		for name := range strings.SplitSeq(s, ",") {
			name = strings.ToLower(name)
			if !slices.Contains(leavable, name) {
				return fmt.Errorf("unknown source %q", name)
			}
			opts.without = append(opts.without, name)
		}
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
	case opts.corpus == "":
		return fmt.Errorf("%w: empty -corpus", errUsage)
	}
	return nil
}
