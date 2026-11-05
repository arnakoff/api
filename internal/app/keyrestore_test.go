// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/arnakoff/api/internal/app"
)

// sealedDumpFake fetches sealed, or fails with err.
type sealedDumpFake struct {
	sealed []byte
	err    error
}

func (f sealedDumpFake) FetchDump(context.Context) ([]byte, error) { return f.sealed, f.err }

// openerFake opens a sealed dump by dropping its "sealed:" prefix, and
// keeps the clear dump it returned to check it is wiped after use.
type openerFake struct {
	opened []byte
	err    error
}

func (o *openerFake) OpenDump(sealed []byte) ([]byte, error) {
	if o.err != nil {
		return nil, o.err
	}
	o.opened = bytes.Clone(bytes.TrimPrefix(sealed, []byte("sealed:")))
	return o.opened, nil
}

// loaderFake keeps a copy of the dump it loaded, or fails with err.
type loaderFake struct {
	loaded []byte
	err    error
}

func (l *loaderFake) LoadDump(_ context.Context, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	l.loaded = bytes.Clone(b)
	return l.err
}

// zeroed reports whether b holds only zero bytes.
func zeroed(b []byte) bool {
	return bytes.Equal(b, make([]byte, len(b)))
}

func TestKeyRestoreLoadsTheOpenedDumpThenWipesIt(t *testing.T) {
	t.Parallel()
	opener, loader := &openerFake{}, &loaderFake{}
	r := app.NewKeyRestore(sealedDumpFake{sealed: []byte("sealed:dump")}, opener, loader)

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(loader.loaded) != "dump" {
		t.Errorf("loaded %q, want %q", loader.loaded, "dump")
	}
	if len(opener.opened) == 0 || !zeroed(opener.opened) {
		t.Errorf("clear dump after Run = %q, want zeroed", opener.opened)
	}
}

func TestKeyRestoreStopsAtTheFirstFailure(t *testing.T) {
	t.Parallel()
	failed := errors.New("failed")
	sealed := sealedDumpFake{sealed: []byte("sealed:dump")}
	tests := map[string]struct {
		fetcher sealedDumpFake
		opener  *openerFake
		loader  *loaderFake
		loaded  bool
	}{
		"fetch": {sealedDumpFake{err: failed}, &openerFake{}, &loaderFake{}, false},
		"open":  {sealed, &openerFake{err: failed}, &loaderFake{}, false},
		"load":  {sealed, &openerFake{}, &loaderFake{err: failed}, true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := app.NewKeyRestore(tt.fetcher, tt.opener, tt.loader).Run(context.Background())
			if !errors.Is(err, failed) {
				t.Fatalf("Run() error = %v, want %v", err, failed)
			}
			if got := tt.loader.loaded != nil; got != tt.loaded {
				t.Errorf("dump loaded = %v, want %v", got, tt.loaded)
			}
			if !zeroed(tt.opener.opened) {
				t.Errorf("clear dump after a failed Run = %q, want zeroed", tt.opener.opened)
			}
		})
	}
}
