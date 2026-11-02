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
	"github.com/arnakoff/api/internal/adapter/reference/hosting"
	"github.com/arnakoff/api/internal/adapter/reference/official"
	"github.com/arnakoff/api/internal/adapter/registry/live"
	"github.com/arnakoff/api/internal/adapter/registry/replay"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/config"
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
	os.Exit(startup.Run("eval", os.LookupEnv, os.Stderr, func(ctx context.Context, cfg config.Config) error {
		return run(ctx, opts, cfg.SnapshotDir(), os.Stdout)
	}))
}

// run scores the corpus against the lists stored in snapshotDir and
// writes the report to out. Online, it records the registries' answers to
// the corpus's replay directory.
func run(ctx context.Context, opts options, snapshotDir string, out io.Writer) error {
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
	lists, err := storedLists(ctx, snapshotDir)
	if err != nil {
		return err
	}
	regs, err := registries(opts, tape, lists)
	if err != nil {
		return err
	}
	score, err := app.Evaluate(ctx, app.NewCheck(regs, clock.System{}), cases)
	if !opts.offline {
		err = errors.Join(err, tape.Save(tapeDir))
	}
	if err != nil {
		return err
	}
	return corpus.WriteReport(out, score, opts.without)
}

// registries returns the lists and the live registries recorded on the
// tape, or the tape's alone offline, less those the run leaves out.
func registries(opts options, tape *replay.Tape, lists app.Registries) (app.Registries, error) {
	var online app.Registries
	if !opts.offline {
		var err error
		if online, err = live.Registries(); err != nil {
			return app.Registries{}, err
		}
	}
	regs := tape.Registries(online)
	regs.Blacklists, regs.Whitelists, regs.Numbering, regs.Officials = lists.Blacklists, lists.Whitelists, lists.Numbering, lists.Officials
	return regs.Without(opts.leftOut), nil
}

// storedLists returns the AMF and ACPR blacklists, the AMF white lists and
// the ARCEP numbering plan of the snapshots stored in dir, as the API
// serves them, and the shipped list of official domains. The ACPR
// blacklist is read whatever the configuration: the corpus measures it
// (testdata/corpus/README.md). A list without a stored snapshot is
// reported unavailable.
func storedLists(ctx context.Context, dir string) (app.Registries, error) {
	disk, err := store.NewDisk(dir)
	if err != nil {
		return app.Registries{}, err
	}
	opts := catalog.Options{ACPR: true}
	s, err := app.NewSync(disk, clock.System{}, catalog.New(catalog.Offline{}, opts).Fetchers()...)
	if err != nil {
		return app.Registries{}, err
	}
	if err := s.Restore(ctx); err != nil {
		return app.Registries{}, err
	}
	officials, err := official.Load()
	if err != nil {
		return app.Registries{}, err
	}
	platforms, err := hosting.Load()
	if err != nil {
		return app.Registries{}, err
	}
	lists := catalog.Lists(opts, s, platforms)
	lists.Officials = officials
	return lists, nil
}
