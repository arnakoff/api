// SPDX-License-Identifier: AGPL-3.0-only

package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
)

// MinKeyBytes is the shortest HMAC key New accepts: 32 bytes, the output
// size of SHA-256, which RFC 2104 (section 3) gives as the minimum
// recommended key length.
const MinKeyBytes = 32

// ErrInvalidConfig means New was given a Config it cannot use.
var ErrInvalidConfig = errors.New("invalid logging config")

// Config configures a logger.
type Config struct {
	// Level is the minimum level written; nil means slog.LevelInfo.
	Level slog.Leveler

	// Key is the secret HMAC key that pseudonymises personal data, at
	// least MinKeyBytes long. Required. New copies it.
	Key []byte
}

// New returns a logger writing one JSON object per line to w, with personal
// data masked and the request identifier of the context added.
func New(w io.Writer, cfg Config) (*slog.Logger, error) {
	if len(cfg.Key) < MinKeyBytes {
		return nil, fmt.Errorf("%w: key is %d bytes, want at least %d", ErrInvalidConfig, len(cfg.Key), MinKeyBytes)
	}
	r := newRedactor(cfg.Key)
	json := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       cfg.Level,
		ReplaceAttr: r.replaceAttr,
	})
	return slog.New(newRequestIDHandler(json)), nil
}
