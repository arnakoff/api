// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"errors"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

func date(tb testing.TB, s string) time.Time {
	tb.Helper()
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		tb.Fatal(err)
	}
	return t
}

func TestParseRegistration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body []byte
		want domain.Registration
	}{
		{
			// The registrar entity has its own, later, registration event.
			name: "AFNIC",
			body: readFixture(t, "afnic-afnic.fr.json"),
			want: domain.Registration{
				Registered:  date(t, "2001-12-10T23:00:00Z"),
				LastChanged: date(t, "2026-06-05T06:59:15.743011Z"),
			},
		},
		{
			name: "Verisign",
			body: readFixture(t, "verisign-verisign.com.json"),
			want: domain.Registration{
				Registered:  date(t, "1995-06-02T04:00:00Z"),
				LastChanged: date(t, "2025-02-07T17:08:30Z"),
			},
		},
		{
			// Deviates from RFC 9083 section 4.5: a full-date without time.
			name: "Register.si date only",
			body: readFixture(t, "register.si-ilirika.si.json"),
			want: domain.Registration{
				Registered:  date(t, "1997-09-02T00:00:00Z"),
				LastChanged: date(t, "2025-03-18T00:00:00Z"),
			},
		},
		{
			name: "no events",
			body: []byte(`{"objectClassName":"domain","ldhName":"exemple.fr"}`),
			want: domain.Registration{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRegistration(tt.body)
			if err != nil {
				t.Fatalf("parseRegistration() error = %v", err)
			}
			if !got.Registered.Equal(tt.want.Registered) || !got.LastChanged.Equal(tt.want.LastChanged) {
				t.Errorf("parseRegistration() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseRegistrationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{"not JSON", `<html>`},
		{"not a domain", `{"objectClassName":"entity","events":[]}`},
		{"invalid date", `{"objectClassName":"domain","events":[{"eventAction":"registration","eventDate":"10/12/2001"}]}`},
		{"invalid date only", `{"objectClassName":"domain","events":[{"eventAction":"registration","eventDate":"1997-13-02"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseRegistration([]byte(tt.body)); !errors.Is(err, ErrResponse) {
				t.Errorf("parseRegistration() error = %v, want ErrResponse", err)
			}
		})
	}
}
