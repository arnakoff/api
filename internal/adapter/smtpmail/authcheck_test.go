// SPDX-License-Identifier: AGPL-3.0-only

package smtpmail_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/smtpmail"
	"github.com/arnakoff/api/internal/app"
)

var _ app.CredentialInstaller = (*smtpmail.AuthCheck)(nil)

// mailKey is a key of the mail application, as the rotation job creates it.
var mailKey = app.Credential{AccessKey: "SCWMAILXXXXXXXXXXXXX", SecretKey: []byte("5e3c0a4e-0000-4000-8000-000000000001")}

// authCheck returns an AuthCheck of srv, trusting its certificate, that
// tries attempts times and counts its pauses in *pauses.
func authCheck(t *testing.T, srv *fakeServer, attempts int, pauses *int) *smtpmail.AuthCheck {
	t.Helper()
	c, err := smtpmail.NewAuthCheck(smtpmail.AuthCheckOptions{
		Addr: srv.addr, Username: "11111111-1111-1111-1111-111111111111", HelloName: "arnakoff.com",
		RootCAs: srv.roots, Attempts: attempts, Pause: time.Minute,
		Sleep: func(context.Context, time.Duration) error { *pauses++; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAuthCheckAuthenticatesWithTheSecretKeyAndSendsNothing(t *testing.T) {
	t.Parallel()
	srv := newFakeServer(t, "")
	var pauses int

	if err := authCheck(t, srv, 3, &pauses).InstallCredential(context.Background(), mailKey); err != nil {
		t.Fatalf("InstallCredential() error = %v", err)
	}

	commands, data := srv.received()
	plain := base64.StdEncoding.EncodeToString([]byte("\x0011111111-1111-1111-1111-111111111111\x00" + string(mailKey.SecretKey)))
	want := []string{"EHLO arnakoff.com", "AUTH PLAIN " + plain, "QUIT"}
	if strings.Join(commands, "|") != strings.Join(want, "|") || data != "" || pauses != 0 {
		t.Errorf("commands %q, data %q, pauses %d; want %q, no mail, no pause", commands, data, pauses, want)
	}
}

func TestAuthCheckTriesAgainThenFailsWithoutTheSecretKey(t *testing.T) {
	t.Parallel()
	srv := newFakeServer(t, "AUTH")
	var pauses int

	err := authCheck(t, srv, 3, &pauses).InstallCredential(context.Background(), mailKey)

	if err == nil || strings.Contains(err.Error(), string(mailKey.SecretKey)) || !strings.Contains(err.Error(), mailKey.AccessKey) {
		t.Fatalf("InstallCredential() error = %v, want a failure naming the access key only", err)
	}
	commands, _ := srv.received()
	auths := 0
	for _, c := range commands {
		if strings.HasPrefix(c, "AUTH") {
			auths++
		}
	}
	if auths != 3 || pauses != 2 {
		t.Errorf("%d AUTH and %d pauses, want 3 and 2", auths, pauses)
	}
}

func TestAuthCheckStopsWhenTheContextIsDone(t *testing.T) {
	t.Parallel()
	srv := newFakeServer(t, "AUTH")
	c, err := smtpmail.NewAuthCheck(smtpmail.AuthCheckOptions{
		Addr: srv.addr, Username: "u", HelloName: "arnakoff.com", RootCAs: srv.roots, Attempts: 5, Pause: time.Minute,
		Sleep: func(context.Context, time.Duration) error { return context.Canceled },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.InstallCredential(context.Background(), mailKey); err == nil {
		t.Error("InstallCredential() = nil, want the failure once the pause is cut short")
	}
}

func TestNewAuthCheckRefusesIncompleteOptions(t *testing.T) {
	t.Parallel()
	for name, opts := range map[string]smtpmail.AuthCheckOptions{
		"no address":       {Username: "u", HelloName: "arnakoff.com"},
		"no port":          {Addr: "smtp.tem.scaleway.com", Username: "u", HelloName: "arnakoff.com"},
		"no username":      {Addr: "smtp.tem.scaleway.com:465", HelloName: "arnakoff.com"},
		"no hello name":    {Addr: "smtp.tem.scaleway.com:465", Username: "u"},
		"negative attempt": {Addr: "smtp.tem.scaleway.com:465", Username: "u", HelloName: "arnakoff.com", Attempts: -1},
	} {
		if _, err := smtpmail.NewAuthCheck(opts); !errors.Is(err, smtpmail.ErrConfig) {
			t.Errorf("%s: NewAuthCheck() = %v, want ErrConfig", name, err)
		}
	}
}
