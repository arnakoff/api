// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func TestWriteKeys(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	var b strings.Builder
	err := writeKeys(&b, []domain.KeyRecord{
		{ID: 1, Environment: domain.KeyLive, Label: "Bot Discord", Hint: "a1B2", CreatedAt: created},
		{ID: 2, Environment: domain.KeyTest, Label: "Essai", Hint: "c3D4", CreatedAt: created, RevokedAt: created.Add(time.Hour)},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "ID  ENV   HINT  CREATED               REVOKED               LABEL\n" +
		"1   live  a1B2  2026-11-02T10:00:00Z  -                     Bot Discord\n" +
		"2   test  c3D4  2026-11-02T10:00:00Z  2026-11-02T11:00:00Z  Essai\n"
	if b.String() != want {
		t.Errorf("writeKeys() =\n%s\nwant\n%s", b.String(), want)
	}
}

func TestWriteIssued(t *testing.T) {
	t.Parallel()
	key, err := domain.ParseAPIKey(sampleKey(t))
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := writeIssued(&b, key, domain.KeyRecord{ID: 7}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), key.String()) || !strings.Contains(b.String(), "7") {
		t.Errorf("writeIssued() = %q, want the key and its identifier", b.String())
	}
}

// sampleKey returns a well-formed test key.
func sampleKey(t *testing.T) string {
	t.Helper()
	k, err := domain.NewAPIKey(domain.KeyTest, strings.NewReader(strings.Repeat("k", 64)))
	if err != nil {
		t.Fatal(err)
	}
	return k.String()
}
