// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/platform/config"
)

func TestSecretNeverWritesItsValue(t *testing.T) {
	t.Parallel()
	s := config.Secret("s3cr3t-value")
	cases := map[string]func() string{
		"%v":  func() string { return fmt.Sprintf("%v", s) },
		"%s":  func() string { return fmt.Sprintf("%s", s) },
		"%x":  func() string { return fmt.Sprintf("%x", s) },
		"%q":  func() string { return fmt.Sprintf("%q", s) },
		"%#v": func() string { return fmt.Sprintf("%#v", s) },
		"%+v in a struct": func() string {
			return fmt.Sprintf("%+v", struct{ Key config.Secret }{s})
		},
		"json": func() string {
			b, err := json.Marshal(struct{ Key config.Secret }{s})
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		},
		"slog": func() string {
			var buf bytes.Buffer
			slog.New(slog.NewJSONHandler(&buf, nil)).Info("x", "key", s)
			return buf.String()
		},
	}
	for name, write := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := write()
			if strings.Contains(got, "s3cr3t") || strings.Contains(got, fmt.Sprintf("%x", []byte(s))) {
				t.Fatalf("output %q leaks the secret", got)
			}
			if !strings.Contains(got, "[redacted]") {
				t.Fatalf("output %q, want it to contain [redacted]", got)
			}
		})
	}
}

func TestSecretBytesReturnsACopy(t *testing.T) {
	t.Parallel()
	s := config.Secret("abc")
	b := s.Bytes()
	b[0] = 'z'
	if got := string(s.Bytes()); got != "abc" {
		t.Fatalf("Bytes after mutating a previous result = %q, want %q", got, "abc")
	}
}
