// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// ErrBackupRunning means a backup was asked while another one runs.
var ErrBackupRunning = errors.New("a backup is already running")

// BackupDumper writes a dump of the key store, which holds personal data:
// the email addresses of the key holders.
type BackupDumper interface {
	Dump(ctx context.Context, w io.Writer) error
}

// BackupSealer encrypts what is written to the returned WriteCloser into w,
// for a key the process does not hold: it can seal a backup, never open
// one. Close writes the end of the sealed data.
type BackupSealer interface {
	Seal(w io.Writer) (io.WriteCloser, error)
}

// BackupArchive keeps a sealed backup under name, out of the host of the
// key store (#75). It can add a backup, never read or remove one.
type BackupArchive interface {
	Keep(ctx context.Context, name string, body []byte) error
}

// backupNameLayout dates a backup in UTC, to the second.
const backupNameLayout = "2006-01-02T150405Z"

// Backup copies the key store, sealed, to an archive, one run at a time.
type Backup struct {
	dumper  BackupDumper
	sealer  BackupSealer
	archive BackupArchive
	clock   Clock
	running sync.Mutex
}

// NewBackup returns a Backup of dumper into archive, sealed by sealer,
// named after the time of clock.
func NewBackup(dumper BackupDumper, sealer BackupSealer, archive BackupArchive, clock Clock) *Backup {
	return &Backup{dumper: dumper, sealer: sealer, archive: archive, clock: clock}
}

// Run dumps the key store, seals the dump in memory and keeps it in the
// archive, then returns its name. The clear dump never leaves the sealer.
// It returns ErrBackupRunning while another Run is in progress.
func (b *Backup) Run(ctx context.Context) (string, error) {
	if !b.running.TryLock() {
		return "", ErrBackupRunning
	}
	defer b.running.Unlock()
	name := "keys/" + b.clock.Now().UTC().Format(backupNameLayout) + ".tar.age"
	body, err := b.sealedDump(ctx)
	if err != nil {
		return "", err
	}
	if err := b.archive.Keep(ctx, name, body); err != nil {
		return "", fmt.Errorf("keep backup %s: %w", name, err)
	}
	return name, nil
}

// sealedDump returns the dump of the key store, sealed.
func (b *Backup) sealedDump(ctx context.Context) ([]byte, error) {
	var sealed bytes.Buffer
	w, err := b.sealer.Seal(&sealed)
	if err != nil {
		return nil, fmt.Errorf("seal backup: %w", err)
	}
	if err := b.dumper.Dump(ctx, w); err != nil {
		return nil, fmt.Errorf("dump key store: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("seal backup: %w", err)
	}
	return sealed.Bytes(), nil
}
