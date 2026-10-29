// SPDX-License-Identifier: AGPL-3.0-only

//go:build !unix

package store

// syncDir does nothing outside Unix: the API runs on Linux, and a local
// run elsewhere does not need crash durability.
func syncDir(string) error {
	return nil
}
