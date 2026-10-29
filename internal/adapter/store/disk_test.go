// SPDX-License-Identifier: AGPL-3.0-only

package store_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/store"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// The store implements the port the sync use case consumes.
var _ app.SnapshotStore = (*store.Disk)(nil)

// fetchedAt is the fixed fetch time of the test snapshots.
var fetchedAt = time.Date(2026, 10, 29, 12, 0, 0, 123456789, time.UTC)

// amfSnapshot returns a small AMF snapshot with a dated newest entry.
func amfSnapshot() domain.Snapshot {
	return domain.Snapshot{
		Source: "AMF",
		Header: []string{"nom", "categorie", "date_inscription"},
		Rows: [][]string{
			{"site-a.example", "Forex", "2026-10-08"},
			{"Société « Ç » \"quoted\"\n", "", "2026-10-28"},
		},
		FetchedAt: fetchedAt,
		Newest:    time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC),
	}
}

// newDisk returns a store in a fresh temporary directory.
func newDisk(t *testing.T) (*store.Disk, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "snapshots")
	d, err := store.NewDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	return d, dir
}

// assertSame fails unless got holds the same snapshot as want.
func assertSame(t *testing.T, got, want domain.Snapshot) {
	t.Helper()
	if !got.FetchedAt.Equal(want.FetchedAt) || !got.Newest.Equal(want.Newest) {
		t.Errorf("times = %v, %v; want %v, %v", got.FetchedAt, got.Newest, want.FetchedAt, want.Newest)
	}
	got.FetchedAt, got.Newest, want.FetchedAt, want.Newest = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
}

func TestDiskLoadsWhatItSaved(t *testing.T) {
	t.Parallel()
	undated := amfSnapshot()
	undated.Newest = time.Time{}
	for name, snap := range map[string]domain.Snapshot{"dated": amfSnapshot(), "undated": undated} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d, _ := newDisk(t)
			if err := d.Save(t.Context(), snap); err != nil {
				t.Fatal(err)
			}
			got, err := d.Load(t.Context(), "AMF")
			if err != nil {
				t.Fatal(err)
			}
			assertSame(t, got, snap)
		})
	}
}

func TestDiskSaveReplacesThePreviousSnapshot(t *testing.T) {
	t.Parallel()
	d, dir := newDisk(t)
	first := amfSnapshot()
	second := amfSnapshot()
	second.Rows = second.Rows[:1]
	second.FetchedAt = fetchedAt.Add(time.Hour)
	for _, s := range []domain.Snapshot{first, second} {
		if err := d.Save(t.Context(), s); err != nil {
			t.Fatal(err)
		}
	}
	got, err := d.Load(t.Context(), "AMF")
	if err != nil {
		t.Fatal(err)
	}
	assertSame(t, got, second)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "amf.snapshot" {
		t.Errorf("directory holds %v, want only amf.snapshot", entries)
	}
}

func TestDiskKeepsEachSourceApart(t *testing.T) {
	t.Parallel()
	d, _ := newDisk(t)
	amf := amfSnapshot()
	acpr := amfSnapshot()
	acpr.Source = "ACPR"
	acpr.Rows = acpr.Rows[:1]
	for _, s := range []domain.Snapshot{amf, acpr} {
		if err := d.Save(t.Context(), s); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []domain.Snapshot{amf, acpr} {
		got, err := d.Load(t.Context(), want.Source)
		if err != nil {
			t.Fatal(err)
		}
		assertSame(t, got, want)
	}
}

func TestDiskLoadWithoutSnapshot(t *testing.T) {
	t.Parallel()
	d, _ := newDisk(t)
	_, err := d.Load(t.Context(), "AMF")
	if !errors.Is(err, app.ErrNoSnapshot) {
		t.Errorf("Load = %v, want ErrNoSnapshot", err)
	}
}

func TestDiskLoadRejectsADamagedFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		damage func([]byte) []byte
		want   error
	}{
		{"flipped byte", func(b []byte) []byte { b[len(b)-10] ^= 1; return b }, store.ErrCorrupt},
		{"truncated", func(b []byte) []byte { return b[:len(b)-5] }, store.ErrCorrupt},
		{"empty", func([]byte) []byte { return nil }, store.ErrCorrupt},
		{"no envelope", func(b []byte) []byte { return b[bytes.IndexByte(b, '\n')+1:] }, store.ErrCorrupt},
		{"other format", func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"arnakoff-snapshot"`), []byte(`"other"`), 1)
		}, store.ErrCorrupt},
		{"other version", func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version":1`), []byte(`"version":2`), 1)
		}, store.ErrVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, dir := newDisk(t)
			if err := d.Save(t.Context(), amfSnapshot()); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "amf.snapshot")
			b, err := os.ReadFile(path) //nolint:gosec // G304: a path under t.TempDir.
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, tt.damage(b), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := d.Load(t.Context(), "AMF"); !errors.Is(err, tt.want) {
				t.Errorf("Load = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDiskLoadRejectsTheSnapshotOfAnotherSource(t *testing.T) {
	t.Parallel()
	d, dir := newDisk(t)
	other := amfSnapshot()
	other.Source = "amf" // same file name, another source
	if err := d.Save(t.Context(), other); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "amf.snapshot")); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Load(t.Context(), "AMF"); !errors.Is(err, store.ErrCorrupt) {
		t.Errorf("Load = %v, want ErrCorrupt", err)
	}
}

func TestDiskFileNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		source domain.Source
		file   string
	}{
		{"AMF", "amf.snapshot"},
		{"AMF listes blanches", "amf-listes-blanches.snapshot"},
		{"../../etc/ARCEP", "etc-arcep.snapshot"},
		{"  Opérateurs ", "op-rateurs.snapshot"},
	}
	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			t.Parallel()
			d, dir := newDisk(t)
			snap := amfSnapshot()
			snap.Source = tt.source
			if err := d.Save(t.Context(), snap); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(dir, tt.file)); err != nil {
				t.Errorf("Save did not write %s: %v", tt.file, err)
			}
		})
	}
}

func TestDiskRejectsASourceWithoutUsableName(t *testing.T) {
	t.Parallel()
	d, _ := newDisk(t)
	for _, source := range []domain.Source{"", "..", "/", "« »"} {
		snap := amfSnapshot()
		snap.Source = source
		if err := d.Save(t.Context(), snap); !errors.Is(err, store.ErrSourceName) {
			t.Errorf("Save(%q) = %v, want ErrSourceName", source, err)
		}
		if _, err := d.Load(t.Context(), source); !errors.Is(err, store.ErrSourceName) {
			t.Errorf("Load(%q) = %v, want ErrSourceName", source, err)
		}
	}
}

func TestDiskStopsOnACanceledContext(t *testing.T) {
	t.Parallel()
	d, dir := newDisk(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := d.Save(ctx, amfSnapshot()); !errors.Is(err, context.Canceled) {
		t.Errorf("Save = %v, want context.Canceled", err)
	}
	if _, err := d.Load(ctx, "AMF"); !errors.Is(err, context.Canceled) {
		t.Errorf("Load = %v, want context.Canceled", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("directory holds %v, want nothing", entries)
	}
}

func TestDiskSaveFailsWhenTheDirectoryIsGone(t *testing.T) {
	t.Parallel()
	d, dir := newDisk(t)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := d.Save(t.Context(), amfSnapshot()); err == nil {
		t.Error("Save = nil, want an error")
	}
}

func TestNewDiskCreatesTheDirectory(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "a", "b")
	if _, err := store.NewDisk(dir); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("Stat(%s) = %v, %v; want a directory", dir, info, err)
	}
}

func TestNewDiskFailsOnAFile(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewDisk(file); err == nil {
		t.Error("NewDisk on a file = nil, want an error")
	}
}
