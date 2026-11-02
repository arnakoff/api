// SPDX-License-Identifier: AGPL-3.0-only

package domain_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func TestDigestAPIKey(t *testing.T) {
	t.Parallel()

	k, err := domain.NewAPIKey(domain.KeyLive, zeros{})
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("0123456789abcdef0123456789abcdef")
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(k.String()))
	var want domain.KeyDigest
	copy(want[:], mac.Sum(nil))

	if got := domain.DigestAPIKey(secret, k); got != want {
		t.Errorf("DigestAPIKey() = %x, want the HMAC-SHA256 of the key, %x", got, want)
	}
	if other := domain.DigestAPIKey([]byte("another secret, as long as needed"), k); other == want {
		t.Error("DigestAPIKey() does not depend on the secret")
	}
}

func TestKeyRecordActive(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	if r := (domain.KeyRecord{CreatedAt: created}); !r.Active() {
		t.Error("Active() = false for a key never revoked")
	}
	if r := (domain.KeyRecord{CreatedAt: created, RevokedAt: created.Add(time.Hour)}); r.Active() {
		t.Error("Active() = true for a revoked key")
	}
}

func TestParseKeyLabel(t *testing.T) {
	t.Parallel()

	if got, err := domain.ParseKeyLabel("  Bot Discord de l'association  "); err != nil || got != "Bot Discord de l'association" {
		t.Errorf("ParseKeyLabel() = %q, %v, want the trimmed label", got, err)
	}
	long := strings.Repeat("é", domain.MaxKeyLabelRunes)
	if _, err := domain.ParseKeyLabel(long); err != nil {
		t.Errorf("ParseKeyLabel(%d runes) error = %v", domain.MaxKeyLabelRunes, err)
	}
	for _, in := range []string{"", "   ", long + "é", "tab\there", "new\nline", "\xff"} {
		if _, err := domain.ParseKeyLabel(in); !errors.Is(err, domain.ErrInvalidKeyLabel) {
			t.Errorf("ParseKeyLabel(%q) error = %v, want ErrInvalidKeyLabel", in, err)
		}
	}
}
