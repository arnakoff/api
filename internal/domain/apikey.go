// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"strings"
)

// ErrInvalidAPIKey means a string is not a well-formed API key: wrong
// prefix, environment, length, alphabet or checksum.
var ErrInvalidAPIKey = errors.New("invalid API key")

// keyAlphabet is base62: digits, upper-case then lower-case ASCII letters.
const keyAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Lengths of the parts of a key after its prefix: 30 random characters
// carry 178 bits, and 6 characters hold any CRC-32 (62^6 > 2^32).
const (
	keyRandomLen   = 30
	keyChecksumLen = 6
	keyHintLen     = 4
)

// keyByteLimit is the largest multiple of 62 a byte holds: a random byte at
// or above it is drawn again, so that every character is equally likely.
const keyByteLimit = 256 - 256%len(keyAlphabet)

// KeyEnvironment tells where an API key is valid: production or the
// staging and development environments. The zero value is none.
type KeyEnvironment int

// The environments of an API key.
const (
	KeyLive KeyEnvironment = iota + 1
	KeyTest
)

// keyEnvironments are the environments a key may name.
var keyEnvironments = [...]KeyEnvironment{KeyLive, KeyTest}

// String returns "live" or "test", the name the prefix of a key carries.
func (e KeyEnvironment) String() string {
	switch e {
	case KeyLive:
		return "live"
	case KeyTest:
		return "test"
	default:
		return ""
	}
}

// ParseKeyEnvironment reads "live" or "test".
func ParseKeyEnvironment(s string) (KeyEnvironment, error) {
	for _, e := range keyEnvironments {
		if s == e.String() {
			return e, nil
		}
	}
	return 0, fmt.Errorf("%w: unknown environment", ErrInvalidAPIKey)
}

// prefix returns "ark_<environment>_", which secret scanners match.
func (e KeyEnvironment) prefix() string {
	return "ark_" + e.String() + "_"
}

// APIKey is an API key: "ark_live_" or "ark_test_", 30 random base62
// characters, then the CRC-32 of everything before it in 6 base62
// characters, which lets a scanner tell a real key from a look-alike
// without asking the API. It is a secret: only Redacted may be logged. The
// zero value is not a valid key.
type APIKey struct {
	env   KeyEnvironment
	value string
}

// NewAPIKey draws a new key of env from random, such as crypto/rand.Reader.
func NewAPIKey(env KeyEnvironment, random io.Reader) (APIKey, error) {
	if env.String() == "" {
		return APIKey{}, fmt.Errorf("%w: unknown environment", ErrInvalidAPIKey)
	}
	body, err := randomBase62(random, keyRandomLen)
	if err != nil {
		return APIKey{}, fmt.Errorf("draw API key: %w", err)
	}
	head := env.prefix() + body
	return APIKey{env: env, value: head + keyChecksum(head)}, nil
}

// ParseAPIKey reads a key as NewAPIKey writes it, exactly: no space, no
// change of case.
func ParseAPIKey(s string) (APIKey, error) {
	env, rest, ok := splitKeyPrefix(s)
	if !ok || len(rest) != keyRandomLen+keyChecksumLen || !isBase62(rest) {
		return APIKey{}, ErrInvalidAPIKey
	}
	head := s[:len(s)-keyChecksumLen]
	if keyChecksum(head) != s[len(head):] {
		return APIKey{}, ErrInvalidAPIKey
	}
	return APIKey{env: env, value: s}, nil
}

// String returns the whole key: a secret, shown once to its holder.
func (k APIKey) String() string {
	return k.value
}

// Environment returns where the key is valid.
func (k APIKey) Environment() KeyEnvironment {
	return k.env
}

// Hint returns the last 4 characters of the key, part of its checksum: they
// tell keys apart in a listing and reveal nothing of the random part.
func (k APIKey) Hint() string {
	if len(k.value) < keyHintLen {
		return ""
	}
	return k.value[len(k.value)-keyHintLen:]
}

// Redacted returns the prefix and the hint, such as "ark_live_…a1B2": safe
// for logs.
func (k APIKey) Redacted() string {
	return k.env.prefix() + "…" + k.Hint()
}

// splitKeyPrefix returns the environment named by the prefix of s and what
// follows it.
func splitKeyPrefix(s string) (KeyEnvironment, string, bool) {
	for _, e := range keyEnvironments {
		if rest, ok := strings.CutPrefix(s, e.prefix()); ok {
			return e, rest, true
		}
	}
	return 0, "", false
}

// randomBase62 draws n base62 characters from random, by rejection sampling.
func randomBase62(random io.Reader, n int) (string, error) {
	out := make([]byte, 0, n)
	buf := make([]byte, n)
	for len(out) < n {
		if _, err := io.ReadFull(random, buf[:n-len(out)]); err != nil {
			return "", err
		}
		for _, b := range buf[:n-len(out)] {
			if int(b) < keyByteLimit {
				out = append(out, keyAlphabet[int(b)%len(keyAlphabet)])
			}
		}
	}
	return string(out), nil
}

// keyChecksum returns the CRC-32 (IEEE) of head in keyChecksumLen base62
// characters, most significant first, padded with '0'.
func keyChecksum(head string) string {
	sum := uint64(crc32.ChecksumIEEE([]byte(head)))
	out := make([]byte, keyChecksumLen)
	for i := keyChecksumLen - 1; i >= 0; i-- {
		out[i] = keyAlphabet[sum%uint64(len(keyAlphabet))]
		sum /= uint64(len(keyAlphabet))
	}
	return string(out)
}

// isBase62 reports whether every byte of s is in keyAlphabet.
func isBase62(s string) bool {
	for i := range len(s) {
		if !strings.ContainsRune(keyAlphabet, rune(s[i])) {
			return false
		}
	}
	return true
}
