// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"time"
)

// defaultCorpus is the corpus directory, relative to the repository root.
const defaultCorpus = "testdata/corpus"

// defaultPprof is the loopback address the profiles are served on.
const defaultPprof = "127.0.0.1:6060"

// options are the command-line flags; see the package doc.
type options struct {
	corpus  string
	latency time.Duration
	pprof   string // empty: no profiles
	targets string // non-empty: write the targets for this base URL and stop
	keys    string // file of the API keys the targets carry
}

// errUsage is a misuse of the flags that package flag does not catch.
var errUsage = errors.New("usage")

// parseFlags reads args, writing usage and errors to w. It returns
// flag.ErrHelp for -h.
func parseFlags(args []string, w io.Writer) (options, error) {
	opts := options{corpus: defaultCorpus, pprof: defaultPprof}
	fs := flag.NewFlagSet("loadtest", flag.ContinueOnError)
	fs.SetOutput(w)
	fs.StringVar(&opts.corpus, "corpus", defaultCorpus, "corpus directory, holding scam.csv, legit.csv and replay/")
	fs.DurationVar(&opts.latency, "latency", 0, "time each replayed registry answer takes")
	fs.StringVar(&opts.pprof, "pprof", defaultPprof, "loopback host:port serving /debug/pprof/, empty for none")
	fs.StringVar(&opts.targets, "targets", "", "write the vegeta targets of the corpus for this loopback base URL, then stop")
	fs.StringVar(&opts.keys, "keys", "", "with -targets: file of API keys, one a line, the targets carry in turn")
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
	case opts.latency < 0:
		return fmt.Errorf("%w: negative -latency", errUsage)
	case opts.pprof != "" && !loopback(opts.pprof):
		return fmt.Errorf("%w: -pprof must be a loopback IP and port", errUsage)
	case opts.targets != "" && !loopbackBase(opts.targets):
		return fmt.Errorf("%w: -targets must be http://<loopback IP>:<port>", errUsage)
	case opts.keys != "" && opts.targets == "":
		return fmt.Errorf("%w: -keys needs -targets", errUsage)
	}
	return nil
}

// loopback reports whether hostport is a loopback IP address and a port:
// the profiles tell how the process runs and are never served further.
func loopback(hostport string) bool {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil || port == "" {
		return false
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

// loopbackBase reports whether s is the base URL of a server on a loopback
// address: the load is never sent to another host.
func loopbackBase(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "http" && u.Path == "" && u.RawQuery == "" && u.User == nil && loopback(u.Host)
}
