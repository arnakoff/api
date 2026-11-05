// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/app"
)

// dumper writes data, or fails with err.
type dumper struct {
	data string
	err  error
	// started, when set, is closed as Dump starts, which then blocks
	// until wait is closed.
	started, wait chan struct{}
}

func (d *dumper) Dump(_ context.Context, w io.Writer) error {
	if d.wait != nil {
		close(d.started)
		<-d.wait
	}
	if d.err != nil {
		return d.err
	}
	_, err := io.WriteString(w, d.data)
	return err
}

// sealer wraps what it seals between "sealed(" and ")", or fails with err.
type sealer struct{ err error }

func (s sealer) Seal(w io.Writer) (io.WriteCloser, error) {
	if s.err != nil {
		return nil, s.err
	}
	if _, err := io.WriteString(w, "sealed("); err != nil {
		return nil, err
	}
	return closer{w}, nil
}

type closer struct{ io.Writer }

func (c closer) Close() error {
	_, err := io.WriteString(c.Writer, ")")
	return err
}

// archive keeps what it is given, or fails with err.
type archive struct {
	err  error
	kept map[string]string
}

func (a *archive) Keep(_ context.Context, name string, body []byte) error {
	if a.err != nil {
		return a.err
	}
	if a.kept == nil {
		a.kept = map[string]string{}
	}
	a.kept[name] = string(body)
	return nil
}

func TestBackupKeepsTheSealedDumpUnderItsDate(t *testing.T) {
	t.Parallel()
	a := &archive{}
	now := time.Date(2026, 11, 5, 3, 30, 7, 0, time.FixedZone("CET", 3600))
	b := app.NewBackup(&dumper{data: "rows"}, sealer{}, a, &clock{now: now})

	name, err := b.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	const want = "keys/2026-11-05T023007Z.tar.age"
	if name != want {
		t.Errorf("name = %q, want %q", name, want)
	}
	if got := a.kept[want]; got != "sealed(rows)" {
		t.Errorf("kept %q, want the sealed dump", got)
	}
}

func TestBackupKeepsNothingOnFailure(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	tests := map[string]struct {
		dumper  *dumper
		sealer  sealer
		archive *archive
	}{
		"dump":    {dumper: &dumper{err: boom}, archive: &archive{}},
		"seal":    {dumper: &dumper{data: "rows"}, sealer: sealer{err: boom}, archive: &archive{}},
		"archive": {dumper: &dumper{data: "rows"}, archive: &archive{err: boom}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b := app.NewBackup(tt.dumper, tt.sealer, tt.archive, &clock{now: time.Now()})
			if _, err := b.Run(context.Background()); !errors.Is(err, boom) {
				t.Errorf("Run = %v, want %v", err, boom)
			}
			if len(tt.archive.kept) != 0 {
				t.Errorf("kept %v after a failure", tt.archive.kept)
			}
		})
	}
}

func TestBackupRunsOneAtATime(t *testing.T) {
	t.Parallel()
	d := &dumper{data: "rows", started: make(chan struct{}), wait: make(chan struct{})}
	b := app.NewBackup(d, sealer{}, &archive{}, &clock{now: time.Now()})
	first := make(chan error, 1)
	go func() {
		_, err := b.Run(context.Background())
		first <- err
	}()
	<-d.started
	if _, err := b.Run(context.Background()); !errors.Is(err, app.ErrBackupRunning) {
		t.Errorf("second Run = %v, want ErrBackupRunning", err)
	}
	close(d.wait)
	if err := <-first; err != nil {
		t.Errorf("first Run: %v", err)
	}
}
