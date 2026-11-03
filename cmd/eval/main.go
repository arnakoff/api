// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/arnakoff/api/internal/adapter/assembly"
	"github.com/arnakoff/api/internal/adapter/corpus"
	"github.com/arnakoff/api/internal/adapter/registry/live"
	"github.com/arnakoff/api/internal/adapter/registry/replay"
	"github.com/arnakoff/api/internal/adapter/source/catalog"
	"github.com/arnakoff/api/internal/adapter/source/listing"
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

// measured returns the sources the corpus measures: the ACPR blacklist is
// read whatever the configuration (testdata/corpus/README.md).
func measured() catalog.Options { return catalog.Options{ACPR: true} }

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
	snaps, err := storedSnapshots(ctx, snapshotDir)
	if err != nil {
		return err
	}
	regs, err := registries(opts, tape, snaps)
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

// registries returns the lists stored in snaps, as the API serves them,
// the shipped list of official domains and the live registries recorded
// on the tape, or the tape's alone offline, less those the run leaves out.
func registries(opts options, tape *replay.Tape, snaps listing.Snapshots) (app.Registries, error) {
	var online app.Registries
	if !opts.offline {
		var err error
		if online, err = live.Registries(); err != nil {
			return app.Registries{}, err
		}
	}
	regs, err := assembly.Lists(measured(), snaps, tape.Registries(online))
	if err != nil {
		return app.Registries{}, err
	}
	return regs.Without(opts.leftOut), nil
}

// storedSnapshots returns the snapshots of the measured sources stored in
// dir. A list without a stored snapshot is reported unavailable.
func storedSnapshots(ctx context.Context, dir string) (*app.Sync, error) {
	disk, err := store.NewDisk(dir)
	if err != nil {
		return nil, err
	}
	s, err := assembly.Sync(measured(), disk)
	if err != nil {
		return nil, err
	}
	if err := s.Restore(ctx); err != nil {
		return nil, err
	}
	return s, nil
}
