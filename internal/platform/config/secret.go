// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"fmt"
	"io"
	"log/slog"
	"slices"
)

// redacted is what a Secret writes in place of its value.
const redacted = "[redacted]"

// Secret is a configuration value that must never reach a log, an error or
// a response. Every fmt verb, encoding/json and log/slog write it as
// "[redacted]"; only Bytes gives the value.
type Secret []byte

// Bytes returns a copy of the secret value.
func (s Secret) Bytes() []byte {
	return slices.Clone(s)
}

// Format implements fmt.Formatter: every verb writes "[redacted]".
func (s Secret) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, redacted)
}

// LogValue implements slog.LogValuer.
func (s Secret) LogValue() slog.Value {
	return slog.StringValue(redacted)
}

// MarshalText implements encoding.TextMarshaler, which encoding/json uses.
func (s Secret) MarshalText() ([]byte, error) {
	return []byte(redacted), nil
}
