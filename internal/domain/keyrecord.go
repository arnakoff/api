// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ErrInvalidKeyLabel means a key label is empty, too long or holds a
// control character.
var ErrInvalidKeyLabel = errors.New("invalid API key label")

// RevokedKeyRetention is how long the record of a revoked API key is kept
// after its revocation, then deleted (ADR 0006).
const RevokedKeyRetention = 365 * 24 * time.Hour

// MaxKeyLabelRunes is the longest label ParseKeyLabel accepts, in runes.
const MaxKeyLabelRunes = 100

// KeyDigest is the HMAC-SHA256 of an API key: what is stored in place of
// the key. Keyed with a secret of the application kept out of the store, a
// copy of the store alone does not let anyone check a guessed key.
type KeyDigest [sha256.Size]byte

// DigestAPIKey returns the KeyDigest of k under secret.
func DigestAPIKey(secret []byte, k APIKey) KeyDigest {
	return hmacSHA256(secret, k.String())
}

// hmacSHA256 returns the HMAC-SHA256 of s under secret.
func hmacSHA256(secret []byte, s string) [sha256.Size]byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(s))
	var d [sha256.Size]byte
	copy(d[:], mac.Sum(nil))
	return d
}

// KeyRecord is what is kept about an issued API key: never the key itself.
type KeyRecord struct {
	// ID identifies the key in the administration command; the store sets
	// it.
	ID int64
	// Digest is the KeyDigest of the key.
	Digest KeyDigest
	// Environment is where the key is valid.
	Environment KeyEnvironment
	// Label tells the operator whom the key was issued to.
	Label string
	// Hint is APIKey.Hint, to tell keys apart in a listing.
	Hint string
	// Email is the address of the account the key was issued to by a
	// login link; zero for a key issued by the operator. It is personal
	// data, deleted with the record (ADR 0006).
	Email Email
	// CreatedAt is when the key was issued.
	CreatedAt time.Time
	// RevokedAt is when the key was revoked; zero while it is active.
	RevokedAt time.Time
}

// Active reports whether the key was never revoked.
func (r KeyRecord) Active() bool {
	return r.RevokedAt.IsZero()
}

// ParseKeyLabel trims the spaces around s and accepts from 1 to
// MaxKeyLabelRunes runes of valid UTF-8 with no control character, which
// would garble a listing.
func ParseKeyLabel(s string) (string, error) {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	if n == 0 || n > MaxKeyLabelRunes || !utf8.ValidString(s) || strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return "", ErrInvalidKeyLabel
	}
	return s, nil
}
