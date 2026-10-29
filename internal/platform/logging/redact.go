// SPDX-License-Identifier: AGPL-3.0-only

package logging

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"slices"
)

// pseudonymBytes is the length the HMAC is truncated to: 128 bits, half of
// SHA-256's output, the shortest truncation RFC 2104 (section 5) allows.
const pseudonymBytes = 16

// fallbackRedacted replaces a Redacted form that could not be computed.
const fallbackRedacted = "***"

// personalData is a value object holding personal data, such as
// domain.Email: Redacted is safe for logs, String is not.
type personalData interface {
	Redacted() string
	String() string
}

// redactor masks personal data in log attributes.
type redactor struct {
	key []byte
}

// newRedactor returns a redactor keyed with a copy of key.
func newRedactor(key []byte) redactor {
	return redactor{key: slices.Clone(key)}
}

// replaceAttr is a slog.HandlerOptions.ReplaceAttr that turns personal data
// into its redacted form and pseudonym and leaves every other attribute.
func (r redactor) replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() != slog.KindAny {
		return a
	}
	v, ok := a.Value.Any().(personalData)
	if !ok {
		return a
	}
	return slog.Attr{Key: a.Key, Value: r.mask(v)}
}

// mask returns the redacted form and the pseudonym of v. A method that
// panics, such as one called on a nil pointer, must not take the log line
// or the process down: v is then written as fallbackRedacted alone.
func (r redactor) mask(v personalData) (masked slog.Value) {
	defer func() {
		if recover() != nil {
			masked = slog.GroupValue(slog.String("redacted", fallbackRedacted))
		}
	}()
	return slog.GroupValue(
		slog.String("redacted", v.Redacted()),
		slog.String("hmac", r.pseudonym(v.String())),
	)
}

// pseudonym returns the truncated HMAC-SHA256 of s, in hex.
func (r redactor) pseudonym(s string) string {
	mac := hmac.New(sha256.New, r.key)
	mac.Write([]byte(s))
	return hex.EncodeToString(mac.Sum(nil)[:pseudonymBytes])
}
