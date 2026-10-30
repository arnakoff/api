// SPDX-License-Identifier: AGPL-3.0-only

package replay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// File names of the tracks.
const (
	domainsFile   = "rdap.json"
	companiesFile = "sirene.json"
	noticesFile   = "bodacc.json"
)

// filePerm is the mode of a saved track: test data, read by anyone who
// clones the repository.
const filePerm = 0o644

// Tape holds the recorded answers of the registries. It is safe for
// concurrent use.
type Tape struct {
	domains   track[registration]
	companies track[company]
	notices   track[[]notice]
}

// Load reads the tape in dir. A missing track file is an empty track.
func Load(dir string) (*Tape, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open tape: %w", err)
	}
	defer func() { _ = root.Close() }() // Read only: a close error changes nothing.
	t := &Tape{}
	err = errors.Join(
		load(root, domainsFile, &t.domains),
		load(root, companiesFile, &t.companies),
		load(root, noticesFile, &t.notices),
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Save writes the tape to dir, replacing its track files.
func (t *Tape) Save(dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open tape: %w", err)
	}
	defer func() { _ = root.Close() }() // WriteFile closes each file it writes.
	return errors.Join(
		save(root, domainsFile, &t.domains),
		save(root, companiesFile, &t.companies),
		save(root, noticesFile, &t.notices),
	)
}

// load reads the track file name of root into t.
func load[T any](root *os.Root, name string, t *track[T]) error {
	b, err := root.ReadFile(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if err := json.Unmarshal(b, &t.entries); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	return nil
}

// save writes t to the track file name of root, keys sorted and indented,
// so that a new recording is a readable diff.
func save[T any](root *os.Root, name string, t *track[T]) error {
	t.mu.Lock()
	b, err := json.MarshalIndent(t.entries, "", "  ")
	t.mu.Unlock()
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	if err := root.WriteFile(name, append(b, '\n'), filePerm); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}
