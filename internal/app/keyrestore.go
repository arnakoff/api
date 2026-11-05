// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// KeyDumpFetcher fetches the sealed dump of the key store to restore.
type KeyDumpFetcher interface {
	FetchDump(ctx context.Context) ([]byte, error)
}

// KeyDumpOpener decrypts a sealed dump, whole, so that a dump altered or
// cut short fails here, before anything is loaded.
type KeyDumpOpener interface {
	OpenDump(sealed []byte) ([]byte, error)
}

// KeyDumpLoader loads a clear dump into an empty key store.
type KeyDumpLoader interface {
	LoadDump(ctx context.Context, dump io.Reader) error
}

// KeyRestore loads a sealed dump of the key store into an empty store,
// from inside the private network of production (#297).
type KeyRestore struct {
	fetcher KeyDumpFetcher
	opener  KeyDumpOpener
	loader  KeyDumpLoader
}

// NewKeyRestore returns a KeyRestore of the dump of fetcher, opened by
// opener, into loader.
func NewKeyRestore(fetcher KeyDumpFetcher, opener KeyDumpOpener, loader KeyDumpLoader) *KeyRestore {
	return &KeyRestore{fetcher: fetcher, opener: opener, loader: loader}
}

// Run fetches the sealed dump, opens it and loads it. The clear dump,
// which holds the email addresses of the key holders, stays in memory and
// is wiped before Run returns.
func (r *KeyRestore) Run(ctx context.Context) error {
	sealed, err := r.fetcher.FetchDump(ctx)
	if err != nil {
		return fmt.Errorf("fetch dump: %w", err)
	}
	dump, err := r.opener.OpenDump(sealed)
	if err != nil {
		return fmt.Errorf("open dump: %w", err)
	}
	defer clear(dump)
	if err := r.loader.LoadDump(ctx, bytes.NewReader(dump)); err != nil {
		return fmt.Errorf("load dump: %w", err)
	}
	return nil
}
