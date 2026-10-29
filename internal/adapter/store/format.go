// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Format name and version written in the envelope (see the package doc).
const (
	formatName    = "arnakoff-snapshot"
	formatVersion = 1
)

// envelope is the first line of a snapshot file.
type envelope struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
	SHA256  string `json:"sha256"`
}

// body is the snapshot itself, after the envelope.
type body struct {
	Source    domain.Source `json:"source"`
	FetchedAt time.Time     `json:"fetched_at"`
	Newest    time.Time     `json:"newest,omitzero"`
	Header    []string      `json:"header"`
	Rows      [][]string    `json:"rows"`
}

// encode returns the file content of s.
func encode(s domain.Snapshot) ([]byte, error) {
	b, err := json.Marshal(body{
		Source:    s.Source,
		FetchedAt: s.FetchedAt.UTC(),
		Newest:    s.Newest.UTC(),
		Header:    s.Header,
		Rows:      s.Rows,
	})
	if err != nil {
		return nil, fmt.Errorf("encode body: %w", err)
	}
	sum := sha256.Sum256(b)
	e, err := json.Marshal(envelope{Format: formatName, Version: formatVersion, SHA256: hex.EncodeToString(sum[:])})
	if err != nil {
		return nil, fmt.Errorf("encode envelope: %w", err)
	}
	return bytes.Join([][]byte{e, b}, []byte("\n")), nil
}

// decode returns the snapshot of source held in the file content b.
func decode(b []byte, source domain.Source) (domain.Snapshot, error) {
	head, rest, ok := bytes.Cut(b, []byte("\n"))
	if !ok {
		return domain.Snapshot{}, fmt.Errorf("%w: no envelope line", ErrCorrupt)
	}
	if err := checkEnvelope(head, rest); err != nil {
		return domain.Snapshot{}, err
	}
	var bd body
	if err := json.Unmarshal(rest, &bd); err != nil {
		return domain.Snapshot{}, fmt.Errorf("%w: body: %w", ErrCorrupt, err)
	}
	if bd.Source != source {
		return domain.Snapshot{}, fmt.Errorf("%w: holds source %q", ErrCorrupt, bd.Source)
	}
	return domain.Snapshot{
		Source:    bd.Source,
		Header:    bd.Header,
		Rows:      bd.Rows,
		FetchedAt: bd.FetchedAt,
		Newest:    bd.Newest,
	}, nil
}

// checkEnvelope reports whether head is a known envelope whose digest is
// that of rest.
func checkEnvelope(head, rest []byte) error {
	var e envelope
	if err := json.Unmarshal(head, &e); err != nil {
		return fmt.Errorf("%w: envelope: %w", ErrCorrupt, err)
	}
	if e.Format != formatName {
		return fmt.Errorf("%w: format %q", ErrCorrupt, e.Format)
	}
	if e.Version != formatVersion {
		return fmt.Errorf("%w: %d", ErrVersion, e.Version)
	}
	sum := sha256.Sum256(rest)
	if e.SHA256 != hex.EncodeToString(sum[:]) {
		return fmt.Errorf("%w: digest mismatch", ErrCorrupt)
	}
	return nil
}
