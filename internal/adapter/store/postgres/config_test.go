// SPDX-License-Identifier: AGPL-3.0-only

package postgres_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/store/postgres"
)

func TestParseConfigAccepts(t *testing.T) {
	t.Parallel()

	for _, url := range []string{
		"postgres://api:pw@db.example.eu:5432/arnakoff?sslmode=verify-full",
		"postgres://api:pw@127.0.0.1:5432/arnakoff?sslmode=disable",
		"postgres://api:pw@localhost/arnakoff?sslmode=disable",
		"postgres://api:pw@[::1]/arnakoff",
		"host=/var/run/postgresql dbname=arnakoff sslmode=disable",
	} {
		if _, err := postgres.ParseConfig(url); err != nil {
			t.Errorf("ParseConfig(%q) error = %v", url, err)
		}
	}
}

func TestParseConfigRejectsUnverifiedTLS(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"", "disable", "allow", "prefer", "require", "verify-ca"} {
		url := "postgres://api:s3cret@db.example.eu/arnakoff" //nolint:gosec // G101: a fake password, to prove errors hide it.
		if mode != "" {
			url += "?sslmode=" + mode
		}
		_, err := postgres.ParseConfig(url)
		if !errors.Is(err, postgres.ErrInsecure) {
			t.Errorf("ParseConfig(sslmode=%q) error = %v, want ErrInsecure", mode, err)
		}
		if err != nil && strings.Contains(err.Error(), "s3cret") {
			t.Errorf("ParseConfig(sslmode=%q) error quotes the password: %v", mode, err)
		}
	}
}

func TestParseConfigRejectsRemoteFallback(t *testing.T) {
	t.Parallel()

	url := "postgres://api:pw@127.0.0.1,db.example.eu/arnakoff?sslmode=disable"
	if _, err := postgres.ParseConfig(url); !errors.Is(err, postgres.ErrInsecure) {
		t.Errorf("ParseConfig(loopback then remote host) error = %v, want ErrInsecure", err)
	}
}

func TestParseConfigErrorHidesURL(t *testing.T) {
	t.Parallel()

	_, err := postgres.ParseConfig("postgres://api:s3cret@db.example.eu:notaport/arnakoff")
	if err == nil {
		t.Fatal("ParseConfig(bad port): no error")
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Errorf("ParseConfig() error quotes the password: %v", err)
	}
}
