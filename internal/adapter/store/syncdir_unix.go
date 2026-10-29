// SPDX-License-Identifier: AGPL-3.0-only

//go:build unix

package store

import "os"

// syncDir flushes the directory entry of a renamed file to disk, so the
// rename survives a crash.
func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
