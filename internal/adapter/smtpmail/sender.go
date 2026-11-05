// SPDX-License-Identifier: AGPL-3.0-only

// Package smtpmail sends the mails of the API, such as login links, through
// an authenticated SMTP server reached over implicit TLS (RFC 8314).
package smtpmail

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"net/textproto"
	"slices"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// ErrConfig means a Config misses a value.
var ErrConfig = errors.New("smtp: incomplete configuration")

// timeout bounds a whole mail, connection to QUIT, on top of the deadline
// of its context.
const timeout = 15 * time.Second

// Config configures a Sender.
type Config struct {
	// Addr is the host and port of the server, which speaks TLS at once,
	// such as "smtp.tem.scaleway.com:465".
	Addr string
	// Username and Password authenticate with AUTH PLAIN, sent over TLS
	// only.
	Username string
	Password []byte
	// From is the sender address.
	From domain.Email
	// RootCAs are the authorities the server's certificate must chain to;
	// nil means those of the system.
	RootCAs *x509.CertPool
	// Now dates the mails; nil means time.Now.
	Now func() time.Time
	// Random draws the message identifiers; nil means crypto/rand.Reader.
	Random io.Reader
}

// Sender sends plain text mails. It implements app.Mailer.
type Sender struct {
	cfg  Config
	host string
}

// New returns the Sender of cfg.
func New(cfg Config) (*Sender, error) {
	host, _, err := net.SplitHostPort(cfg.Addr)
	switch {
	case err != nil || host == "":
		return nil, fmt.Errorf("%w: address is not host:port", ErrConfig)
	case cfg.Username == "" || len(cfg.Password) == 0:
		return nil, fmt.Errorf("%w: username and password required", ErrConfig)
	case cfg.From.IsZero():
		return nil, fmt.Errorf("%w: sender address required", ErrConfig)
	}
	cfg.Password = slices.Clone(cfg.Password)
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Random == nil {
		cfg.Random = rand.Reader
	}
	return &Sender{cfg: cfg, host: host}, nil
}

// Send sends to to the plain text mail of subject and body. Its errors
// never quote the address, nor the server's replies, which may.
func (s *Sender) Send(ctx context.Context, to domain.Email, subject, body string) error {
	msg, err := s.message(to, subject, body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	c, err := s.dial(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	return s.deliver(c, to, msg)
}

// dial connects to the server over TLS, verifying its certificate, and
// returns a client bound by the deadline of ctx.
func (s *Sender) dial(ctx context.Context) (*smtp.Client, error) {
	return dialTLS(ctx, s.cfg.Addr, s.host, s.cfg.RootCAs)
}

// dialTLS connects to the server addr of name host over TLS, verifying its
// certificate against roots, those of the system when nil, and returns a
// client bound by the deadline of ctx.
func dialTLS(ctx context.Context, addr, host string, roots *x509.CertPool) (*smtp.Client, error) {
	d := &tls.Dialer{Config: &tls.Config{ServerName: host, RootCAs: roots, MinVersion: tls.VersionTLS12}}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("smtp connect: %w", err)
	}
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("smtp deadline: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, replyError("greeting", err)
	}
	return c, nil
}

// deliver authenticates on c and sends msg to to, then quits.
func (s *Sender) deliver(c *smtp.Client, to domain.Email, msg []byte) error {
	if err := c.Hello(s.cfg.From.Domain().String()); err != nil {
		return replyError("EHLO", err)
	}
	if err := c.Auth(smtp.PlainAuth("", s.cfg.Username, string(s.cfg.Password), s.host)); err != nil {
		return replyError("AUTH", err)
	}
	if err := c.Mail(s.cfg.From.String()); err != nil {
		return replyError("MAIL", err)
	}
	if err := c.Rcpt(to.String()); err != nil {
		return replyError("RCPT", err)
	}
	w, err := c.Data()
	if err != nil {
		return replyError("DATA", err)
	}
	if _, err := w.Write(msg); err != nil {
		return replyError("DATA", err)
	}
	if err := w.Close(); err != nil {
		return replyError("DATA", err)
	}
	return replyError("QUIT", c.Quit())
}

// replyError returns the error of step, keeping of a server reply only its
// code: its text may quote the address. Nil stays nil.
func replyError(step string, err error) error {
	if err == nil {
		return nil
	}
	var reply *textproto.Error
	if errors.As(err, &reply) {
		return fmt.Errorf("smtp %s: server replied %d", step, reply.Code)
	}
	// Errors of net/smtp itself and of the connection quote no address.
	return fmt.Errorf("smtp %s: %w", step, err)
}

// message returns the mail to to, with a body that never holds a lone CR
// or LF in a header: the address comes from domain.ParseEmail, the
// subject is encoded.
func (s *Sender) message(to domain.Email, subject, body string) ([]byte, error) {
	if strings.ContainsAny(to.String(), "\r\n") {
		return nil, errors.New("smtp: address holds a line break")
	}
	id, err := s.messageID()
	if err != nil {
		return nil, err
	}
	return compose(header{
		from: s.cfg.From.String(), to: to.String(), subject: subject,
		date: s.cfg.Now(), id: id,
	}, body)
}

// messageID returns a new Message-ID under the sender's domain.
func (s *Sender) messageID() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(s.cfg.Random, b); err != nil {
		return "", fmt.Errorf("smtp message id: %w", err)
	}
	return fmt.Sprintf("<%x@%s>", b, s.cfg.From.Domain()), nil
}
