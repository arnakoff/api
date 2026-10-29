// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// Store errors, checked with errors.Is.
var (
	ErrCorrupt    = errors.New("corrupt snapshot file")
	ErrVersion    = errors.New("unsupported snapshot format version")
	ErrSourceName = errors.New("source name gives no file name")
)

// dirPerm is the mode of the store directory when NewDisk creates it.
const dirPerm = 0o750

// Disk stores one snapshot file per source in a directory. It implements
// app.SnapshotStore and is safe for concurrent use, also across processes.
type Disk struct {
	dir string
}

// NewDisk returns a Disk storing its files in dir, created if missing.
func NewDisk(dir string) (*Disk, error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("snapshot store %s: %w", dir, err)
	}
	return &Disk{dir: dir}, nil
}

// Save durably replaces the stored snapshot of s.Source: readers see the
// previous file or the new one, never part of one.
func (d *Disk) Save(ctx context.Context, s domain.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, err := fileName(s.Source)
	if err != nil {
		return err
	}
	b, err := encode(s)
	if err != nil {
		return fmt.Errorf("save %s: %w", s.Source, err)
	}
	if err := d.replace(name, b); err != nil {
		return fmt.Errorf("save %s: %w", s.Source, err)
	}
	return nil
}

// Load returns the stored snapshot of source, app.ErrNoSnapshot when there is
// none, or ErrCorrupt or ErrVersion when its file does not check out.
func (d *Disk) Load(ctx context.Context, source domain.Source) (domain.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return domain.Snapshot{}, err
	}
	name, err := fileName(source)
	if err != nil {
		return domain.Snapshot{}, err
	}
	//nolint:gosec // G304: fileName keeps name inside d.dir (no separator, no dot segment).
	b, err := os.ReadFile(filepath.Join(d.dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return domain.Snapshot{}, fmt.Errorf("load %s: %w", source, app.ErrNoSnapshot)
	}
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("load %s: %w", source, err)
	}
	s, err := decode(b, source)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("load %s from %s: %w", source, name, err)
	}
	return s, nil
}

// replace writes b to a temporary file, syncs it, renames it to name and
// syncs the directory. The temporary file is removed on failure.
func (d *Disk) replace(name string, b []byte) (err error) {
	tmp, err := os.CreateTemp(d.dir, name+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := tmp.Write(b); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(d.dir, name)); err != nil {
		return err
	}
	return syncDir(d.dir)
}
