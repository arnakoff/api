// SPDX-License-Identifier: AGPL-3.0-only

package smtpmail_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/arnakoff/api/internal/adapter/smtpmail"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

var _ app.Mailer = (*smtpmail.Sender)(nil)

// sent is the date of the mails of these tests.
var sent = time.Date(2026, 11, 3, 14, 5, 0, 0, time.UTC)

const (
	subject = "Votre lien pour obtenir une clé API Arnakoff"
	body    = "Bonjour,\n\nOuvrez ce lien : https://arnakoff.com/cle#abc\n\nArnakoff — https://arnakoff.com\n"
)

func mustEmail(t *testing.T, s string) domain.Email {
	t.Helper()
	e, err := domain.ParseEmail(s)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// sender returns a Sender to srv, trusting its certificate.
func sender(t *testing.T, srv *fakeServer) *smtpmail.Sender {
	t.Helper()
	s, err := smtpmail.New(smtpmail.Config{
		Addr: srv.addr, Username: "contact@arnakoff.com", Password: []byte("pw"),
		From: mustEmail(t, "contact@arnakoff.com"), RootCAs: srv.roots,
		Now: func() time.Time { return sent }, Random: bytes.NewReader(bytes.Repeat([]byte{7}, 64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSendAuthenticatesAndSendsThePlainTextMail(t *testing.T) {
	t.Parallel()
	srv := newFakeServer(t, "")

	if err := sender(t, srv).Send(t.Context(), mustEmail(t, "Dev@Exemple.fr"), subject, body); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	cmds, data := srv.received()
	auth := base64.StdEncoding.EncodeToString([]byte("\x00contact@arnakoff.com\x00pw"))
	want := []string{"EHLO arnakoff.com", "AUTH PLAIN " + auth, "MAIL FROM:<contact@arnakoff.com>", "RCPT TO:<Dev@exemple.fr>", "DATA", "QUIT"}
	if strings.Join(cmds, "\n") != strings.Join(want, "\n") {
		t.Errorf("commands =\n%s\nwant\n%s", strings.Join(cmds, "\n"), strings.Join(want, "\n"))
	}
	checkMessage(t, data)
}

func checkMessage(t *testing.T, data string) {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(data))
	if err != nil {
		t.Fatalf("message: %v\n%s", err, data)
	}
	h := msg.Header
	dec := new(mime.WordDecoder)
	gotSubject, err := dec.DecodeHeader(h.Get("Subject"))
	if err != nil || gotSubject != subject {
		t.Errorf("Subject = %q (%v), want %q", gotSubject, err, subject)
	}
	for field, want := range map[string]string{
		"From": "Arnakoff <contact@arnakoff.com>", "To": "<Dev@exemple.fr>", "Date": "Tue, 03 Nov 2026 14:05:00 +0000",
		"Content-Type": "text/plain; charset=utf-8", "Content-Transfer-Encoding": "quoted-printable",
		"MIME-Version": "1.0", "Auto-Submitted": "auto-generated",
	} {
		if got := h.Get(field); got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}
	if id := h.Get("Message-Id"); !strings.HasPrefix(id, "<") || !strings.HasSuffix(id, "@arnakoff.com>") {
		t.Errorf("Message-ID = %q, want <…@arnakoff.com>", id)
	}
	raw, err := io.ReadAll(quotedprintable.NewReader(msg.Body))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.ReplaceAll(string(raw), "\r\n", "\n"); got != body {
		t.Errorf("body = %q, want %q", got, body)
	}
}

func TestSendFailsWithoutQuotingTheAddress(t *testing.T) {
	t.Parallel()
	for _, step := range []string{"AUTH", "MAIL", "RCPT", "DATA"} {
		srv := newFakeServer(t, step)
		err := sender(t, srv).Send(t.Context(), mustEmail(t, "dev@exemple.fr"), subject, body)
		if err == nil {
			t.Errorf("%s refused: no error", step)
			continue
		}
		if strings.Contains(err.Error(), "dev@") || !strings.Contains(err.Error(), "550") {
			t.Errorf("%s refused: error %q, want the reply code without the address", step, err)
		}
	}
}

func TestSendRefusesAnUntrustedServer(t *testing.T) {
	t.Parallel()
	srv := newFakeServer(t, "")
	s, err := smtpmail.New(smtpmail.Config{
		Addr: srv.addr, Username: "u", Password: []byte("pw"), From: mustEmail(t, "contact@arnakoff.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Send(t.Context(), mustEmail(t, "dev@exemple.fr"), subject, body); err == nil {
		t.Fatal("Send() to a server of an untrusted certificate: no error")
	}
	if cmds, _ := srv.received(); len(cmds) != 0 {
		t.Errorf("commands sent = %v, want none before the certificate is trusted", cmds)
	}
}

func TestSendGivesUpWhenTheServerHangs(t *testing.T) {
	t.Parallel()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		c, err := ln.Accept()
		if err == nil {
			t.Cleanup(func() { _ = c.Close() })
		}
	}()
	s, err := smtpmail.New(smtpmail.Config{
		Addr: ln.Addr().String(), Username: "u", Password: []byte("pw"), From: mustEmail(t, "contact@arnakoff.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = s.Send(ctx, mustEmail(t, "dev@exemple.fr"), subject, body)
	if err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("Send() = %v after %v, want an error by the deadline", err, time.Since(start))
	}
}

func TestNewRefusesAnIncompleteConfig(t *testing.T) {
	t.Parallel()
	from := mustEmail(t, "contact@arnakoff.com")
	for name, cfg := range map[string]smtpmail.Config{
		"no port":     {Addr: "smtp.mail.ovh.net", Username: "u", Password: []byte("pw"), From: from},
		"no username": {Addr: "smtp.mail.ovh.net:465", Password: []byte("pw"), From: from},
		"no password": {Addr: "smtp.mail.ovh.net:465", Username: "u", From: from},
		"no sender":   {Addr: "smtp.mail.ovh.net:465", Username: "u", Password: []byte("pw")},
	} {
		if _, err := smtpmail.New(cfg); !errors.Is(err, smtpmail.ErrConfig) {
			t.Errorf("%s: error = %v, want ErrConfig", name, err)
		}
	}
}
