// SPDX-License-Identifier: AGPL-3.0-only

package logging_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/logging"
)

// secret stands for a domain value object holding personal data, such as
// domain.Email, which platform may not import.
type secret struct {
	raw, redacted string
}

func (s secret) String() string   { return s.raw }
func (s secret) Redacted() string { return s.redacted }

// pointerSecret has pointer methods, so a nil *pointerSecret panics when
// called.
type pointerSecret struct {
	raw string
}

func (s *pointerSecret) String() string   { return s.raw }
func (s *pointerSecret) Redacted() string { return "***" + s.raw[len(s.raw)-2:] }

// wantHMAC is the pseudonym the logger must give s under testKey: the first
// 16 bytes of HMAC-SHA256, in hex.
func wantHMAC(key []byte, s string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(s))
	return hex.EncodeToString(mac.Sum(nil)[:16])
}

// field reads the group at path in a decoded log line.
func field(t *testing.T, line map[string]any, path ...string) map[string]any {
	t.Helper()
	cur := line
	for _, k := range path {
		next, ok := cur[k].(map[string]any)
		if !ok {
			t.Fatalf("%s is %v, want an object", strings.Join(path, "."), cur[k])
		}
		cur = next
	}
	return cur
}

func TestLoggerRedactsPersonalData(t *testing.T) {
	t.Parallel()
	email := secret{raw: "jean.dupont@exemple.fr", redacted: "***@exemple.fr"}

	tests := []struct {
		name string
		log  func(*slog.Logger)
		path []string
	}{
		{"attr", func(l *slog.Logger) { l.Info("check", slog.Any("email", email)) }, []string{"email"}},
		{"pointer", func(l *slog.Logger) { l.Info("check", slog.Any("email", &email)) }, []string{"email"}},
		{"key-value", func(l *slog.Logger) { l.Info("check", "email", email) }, []string{"email"}},
		{"with", func(l *slog.Logger) { l.With("email", email).Info("check") }, []string{"email"}},
		{"group", func(l *slog.Logger) { l.Info("check", slog.Group("input", "email", email)) }, []string{"input", "email"}},
		{"with group", func(l *slog.Logger) { l.WithGroup("input").Info("check", "email", email) }, []string{"input", "email"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			logger, buf := newTestLogger(t, logging.Config{})

			tt.log(logger)

			if strings.Contains(buf.String(), email.raw) {
				t.Fatalf("log output holds the raw value: %s", buf)
			}
			got := field(t, only(t, buf), tt.path...)
			if got["redacted"] != email.redacted || got["hmac"] != wantHMAC(testKey, email.raw) {
				t.Errorf("%s = %v, want redacted %q and hmac %q", strings.Join(tt.path, "."), got, email.redacted, wantHMAC(testKey, email.raw))
			}
		})
	}
}

func TestLoggerHMACIsStableAndKeyed(t *testing.T) {
	t.Parallel()
	otherKey := []byte(strings.Repeat("k", logging.MinKeyBytes))
	hmacOf := func(key []byte, s string) any {
		logger, buf := newTestLogger(t, logging.Config{Key: key})
		logger.Info("check", "v", secret{raw: s, redacted: "***"})
		return field(t, only(t, buf), "v")["hmac"]
	}

	first := hmacOf(testKey, "+33612345678")
	if again := hmacOf(testKey, "+33612345678"); again != first {
		t.Errorf("hmac changed for the same value and key: %v then %v", first, again)
	}
	if other := hmacOf(testKey, "+33612345679"); other == first {
		t.Errorf("hmac is the same for two values: %v", first)
	}
	if other := hmacOf(otherKey, "+33612345678"); other == first {
		t.Errorf("hmac is the same under two keys: %v", first)
	}
}

func TestNewCopiesKey(t *testing.T) {
	t.Parallel()
	key := append([]byte(nil), testKey...)
	logger, buf := newTestLogger(t, logging.Config{Key: key})
	key[0] ^= 0xff

	logger.Info("check", "v", secret{raw: "FR1420041010050500013M02606", redacted: "FR***2606"})

	if got := field(t, only(t, buf), "v")["hmac"]; got != wantHMAC(testKey, "FR1420041010050500013M02606") {
		t.Errorf("hmac = %v, want the one under the key given to New", got)
	}
}

func TestLoggerSurvivesPanickingValue(t *testing.T) {
	t.Parallel()
	logger, buf := newTestLogger(t, logging.Config{})
	var nilSecret *pointerSecret

	logger.Info("check", "v", nilSecret)

	got := field(t, only(t, buf), "v")
	if got["redacted"] != "***" || got["hmac"] != nil {
		t.Errorf("v = %v, want redacted *** and no hmac", got)
	}
}

func TestLoggerLeavesOtherValues(t *testing.T) {
	t.Parallel()
	logger, buf := newTestLogger(t, logging.Config{})

	logger.Info("check", "source", "amf", "count", 3)

	got := only(t, buf)
	if got["source"] != "amf" || got["count"] != 3.0 {
		t.Errorf("log line = %v, want source amf and count 3", got)
	}
}
