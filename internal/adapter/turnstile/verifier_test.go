// SPDX-License-Identifier: AGPL-3.0-only

package turnstile_test

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/arnakoff/api/internal/adapter/turnstile"
)

// fixtures are the response examples of the siteverify documentation,
// https://developers.cloudflare.com/turnstile/get-started/server-side-validation/
// (read on 02/11/2026). Their hostname is example.com.
var fixtures = os.DirFS("../../../testdata/turnstile")

const (
	secret   = "0x4AAAAAAA-s3cr3t" //nolint:gosec // G101: a fake secret of the tests.
	hostname = "example.com"
)

// submitter is a fake *httpclient.Client: it answers answer or err and
// records what it was sent.
type submitter struct {
	answer []byte
	err    error
	calls  int
	url    string
	ct     string
	form   url.Values
}

func (s *submitter) Submit(_ context.Context, rawURL, contentType string, body []byte) ([]byte, error) {
	s.calls++
	s.url, s.ct = rawURL, contentType
	s.form, _ = url.ParseQuery(string(body))
	return s.answer, s.err
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := fs.ReadFile(fixtures, name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func newVerifier(s *submitter) *turnstile.Verifier {
	return turnstile.New(s, []byte(secret), hostname)
}

func TestVerifyPostsTheTokenToSiteverify(t *testing.T) {
	t.Parallel()
	s := &submitter{answer: fixture(t, "success.json")}

	ok, err := newVerifier(s).Verify(t.Context(), "tok3n")
	if !ok || err != nil {
		t.Fatalf("Verify() = %t, %v, want true, nil", ok, err)
	}
	if s.url != turnstile.SiteverifyURL || s.ct != "application/x-www-form-urlencoded" {
		t.Errorf("sent to %s as %s, want the form to %s", s.url, s.ct, turnstile.SiteverifyURL)
	}
	want := url.Values{"secret": {secret}, "response": {"tok3n"}}
	if s.form.Encode() != want.Encode() {
		t.Errorf("form = %v, want %v", s.form, want)
	}
}

func TestVerifyRefusesWithoutAsking(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, token string }{
		{"empty token", ""},
		{"token over 2048 characters", strings.Repeat("a", 2049)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &submitter{answer: fixture(t, "success.json")}
			ok, err := newVerifier(s).Verify(t.Context(), tt.token)
			if ok || err != nil || s.calls != 0 {
				t.Errorf("Verify() = %t, %v after %d calls, want false, nil and no call", ok, err, s.calls)
			}
		})
	}
}

func TestVerifyRefusesTheTokensSiteverifyRefuses(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, answer string }{
		{"documented failure", string(fixture(t, "failure.json"))},
		{"replayed", `{"success":false,"error-codes":["timeout-or-duplicate"]}`},
		{"missing", `{"success":false,"error-codes":["missing-input-response"]}`},
		{"other site", `{"success":true,"hostname":"evil.example","error-codes":[]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok, err := newVerifier(&submitter{answer: []byte(tt.answer)}).Verify(t.Context(), "tok3n")
			if ok || err != nil {
				t.Errorf("Verify() = %t, %v, want false, nil", ok, err)
			}
		})
	}
}

func TestVerifyFailsWhenTheTokenCannotBeChecked(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		s    *submitter
	}{
		{"transport", &submitter{err: errors.New("connection refused")}},
		{"not JSON", &submitter{answer: []byte("<html>")}},
		{"bad secret", &submitter{answer: []byte(`{"success":false,"error-codes":["invalid-input-secret"]}`)}},
		{"upstream error", &submitter{answer: []byte(`{"success":false,"error-codes":["internal-error"]}`)}},
		{"no code", &submitter{answer: []byte(`{"success":false,"error-codes":[]}`)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok, err := newVerifier(tt.s).Verify(t.Context(), "tok3n")
			if ok || err == nil {
				t.Fatalf("Verify() = %t, %v, want false and an error", ok, err)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "tok3n") {
				t.Errorf("error %q quotes the secret or the token", err)
			}
		})
	}
}

func TestNewLiveRefusesATokenWithoutCallingOut(t *testing.T) {
	t.Parallel()
	v, err := turnstile.NewLive([]byte(secret), hostname)
	if err != nil {
		t.Fatalf("NewLive() error = %v", err)
	}
	if ok, err := v.Verify(t.Context(), ""); ok || err != nil {
		t.Errorf("Verify(empty) = %t, %v, want false, nil", ok, err)
	}
}
