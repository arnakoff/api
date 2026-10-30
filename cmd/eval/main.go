// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/arnakoff/api/internal/adapter/corpus"
	"github.com/arnakoff/api/internal/adapter/registry/bodacc"
	"github.com/arnakoff/api/internal/adapter/registry/rdap"
	"github.com/arnakoff/api/internal/adapter/registry/replay"
	"github.com/arnakoff/api/internal/adapter/registry/sirene"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/config"
	"github.com/arnakoff/api/internal/platform/httpclient"
	"github.com/arnakoff/api/internal/platform/startup"
)

// exitUsage is the exit code of a misuse of the flags.
const exitUsage = 2

// tapeDirPerm is the mode of a replay directory created by a first
// recording: test data, read by anyone who clones the repository.
const tapeDirPerm = 0o755

func main() {
	opts, err := parseFlags(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		os.Exit(exitUsage)
	}
	os.Exit(startup.Run("eval", os.LookupEnv, os.Stderr, func(ctx context.Context, _ config.Config) error {
		return run(ctx, opts, os.Stdout)
	}))
}

// run scores the corpus and writes the report to out. Online, it records
// the registries' answers to the corpus's replay directory.
func run(ctx context.Context, opts options, out io.Writer) error {
	cases, err := corpus.Load(opts.corpus)
	if err != nil {
		return err
	}
	tapeDir := filepath.Join(opts.corpus, "replay")
	if err := os.MkdirAll(tapeDir, tapeDirPerm); err != nil {
		return err
	}
	tape, err := replay.Load(tapeDir)
	if err != nil {
		return err
	}
	regs, err := registries(opts, tape)
	if err != nil {
		return err
	}
	score, err := app.Evaluate(ctx, app.NewCheck(regs), cases)
	if !opts.offline {
		err = errors.Join(err, tape.Save(tapeDir))
	}
	if err != nil {
		return err
	}
	return corpus.WriteReport(out, score, opts.without)
}

// registries returns the live registries recorded on the tape, or the
// tape's alone offline, less those the run leaves out.
func registries(opts options, tape *replay.Tape) (app.Registries, error) {
	var live app.Registries
	if !opts.offline {
		var err error
		if live, err = liveRegistries(); err != nil {
			return app.Registries{}, err
		}
	}
	return tape.Registries(live).Without(opts.leftOut), nil
}

// liveRegistries returns the RDAP, SIRENE and BODACC clients, each with an
// HTTP client allowed its own hosts only.
func liveRegistries() (app.Registries, error) {
	iana, err := httpclient.New(httpclient.Config{AllowedHosts: rdap.BootstrapHosts()})
	if err != nil {
		return app.Registries{}, err
	}
	domains, err := rdap.New(iana, rdap.BootstrapURL, func(hosts []string) (rdap.Getter, error) {
		c, err := httpclient.New(httpclient.Config{AllowedHosts: hosts})
		if err != nil {
			return nil, err
		}
		return c, nil
	})
	if err != nil {
		return app.Registries{}, err
	}
	sc, err := httpclient.New(httpclient.Config{AllowedHosts: sirene.Hosts()})
	if err != nil {
		return app.Registries{}, err
	}
	companies, err := sirene.New(sc, sirene.BaseURL)
	if err != nil {
		return app.Registries{}, err
	}
	bc, err := httpclient.New(httpclient.Config{AllowedHosts: bodacc.Hosts()})
	if err != nil {
		return app.Registries{}, err
	}
	notices, err := bodacc.New(bc, bodacc.BaseURL)
	if err != nil {
		return app.Registries{}, err
	}
	return app.Registries{Domains: domains, Companies: companies, Notices: notices}, nil
}
