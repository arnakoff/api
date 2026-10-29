// SPDX-License-Identifier: AGPL-3.0-only

package adapter_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/logging"
)

// TestLoggerNeverWritesPersonalData logs the domain's personal data value
// objects the ways an adapter would and fails if any raw value, or a
// recognisable part of it, reaches the output.
func TestLoggerNeverWritesPersonalData(t *testing.T) {
	t.Parallel()
	email, err := domain.ParseEmail("Jean.Dupont@exemple.fr")
	if err != nil {
		t.Fatal(err)
	}
	phone, err := domain.ParsePhone("06 12 34 56 78")
	if err != nil {
		t.Fatal(err)
	}
	iban, err := domain.ParseIBAN("FR14 2004 1010 0505 0001 3M02 606")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	logger, err := logging.New(&buf, logging.Config{Key: bytes.Repeat([]byte{0x42}, logging.MinKeyBytes)})
	if err != nil {
		t.Fatal(err)
	}
	ctx := logging.WithRequestID(context.Background(), "req-1")
	logger.InfoContext(ctx, "check", "email", email, "phone", phone, "iban", iban)
	logger.With("email", &email).WarnContext(ctx, "lookup failed", slog.Group("input", "phone", &phone, "iban", &iban))
	logger.WithGroup("check").ErrorContext(ctx, "lookup failed", "iban", iban)

	out := buf.String()
	for _, raw := range []string{
		email.String(), email.Local(),
		phone.String(), "612345678",
		iban.String(), "20041010050500013M026",
	} {
		if strings.Contains(out, raw) {
			t.Errorf("log output holds %q:\n%s", raw, out)
		}
	}
	for _, redacted := range []string{email.Redacted(), phone.Redacted(), iban.Redacted()} {
		if !strings.Contains(out, redacted) {
			t.Errorf("log output lacks redacted form %q:\n%s", redacted, out)
		}
	}
}
