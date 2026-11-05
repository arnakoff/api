// SPDX-License-Identifier: AGPL-3.0-only

package smtpmail

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/clock"
	"github.com/arnakoff/api/internal/platform/defaults"
)

// Defaults of AuthCheckOptions. How long a new key of Scaleway IAM takes to
// work on Transactional Email is not documented (« à vérifier »): as for
// the bucket (ADR 0007), six tries a minute apart leave it five minutes.
const (
	defaultAuthCheckAttempts = 6
	defaultAuthCheckPause    = time.Minute
)

// AuthCheckOptions configures an AuthCheck.
type AuthCheckOptions struct {
	// Addr is the host and port of the server, which speaks TLS at once,
	// such as "smtp.tem.scaleway.com:465"; Username is the user a key
	// authenticates as, its secret key being the password.
	Addr, Username string
	// HelloName is the domain sent with EHLO, such as "arnakoff.com".
	HelloName string
	// RootCAs are the authorities the server's certificate must chain to;
	// nil means those of the system.
	RootCAs *x509.CertPool
	// Attempts is the number of authentications tried, 6 when zero; Pause
	// the wait between two of them, a minute when zero.
	Attempts int
	Pause    time.Duration
	// Sleep waits d, or returns the error of ctx once it is done; the
	// system clock when nil.
	Sleep func(ctx context.Context, d time.Duration) error
}

// AuthCheck checks that a new key of the mail application authenticates
// with the SMTP server before it is put in service (#295). It implements
// app.CredentialInstaller, sends no mail and changes nothing.
type AuthCheck struct {
	opts AuthCheckOptions
	host string
}

// NewAuthCheck returns an AuthCheck of opts, or ErrConfig.
func NewAuthCheck(opts AuthCheckOptions) (*AuthCheck, error) {
	err := errors.Join(
		defaults.Set("attempts", &opts.Attempts, defaultAuthCheckAttempts),
		defaults.Set("pause", &opts.Pause, defaultAuthCheckPause),
	)
	host, _, splitErr := net.SplitHostPort(opts.Addr)
	switch {
	case err != nil:
		return nil, fmt.Errorf("%w: %w", ErrConfig, err)
	case splitErr != nil || host == "":
		return nil, fmt.Errorf("%w: address is not host:port", ErrConfig)
	case opts.Username == "" || opts.HelloName == "":
		return nil, fmt.Errorf("%w: username and hello name required", ErrConfig)
	case opts.Sleep == nil:
		opts.Sleep = clock.System{}.Sleep
	}
	return &AuthCheck{opts: opts, host: host}, nil
}

// InstallCredential authenticates with the secret key of c, until it
// succeeds or the attempts are over. Its errors name the access key of c,
// never its secret key.
func (a *AuthCheck) InstallCredential(ctx context.Context, c app.Credential) error {
	for attempt := 1; ; attempt++ {
		err := a.authenticate(ctx, c.SecretKey)
		if err == nil {
			return nil
		}
		if attempt == a.opts.Attempts || a.opts.Sleep(ctx, a.opts.Pause) != nil {
			return fmt.Errorf("key %s does not authenticate with %s: %w", c.AccessKey, a.opts.Addr, err)
		}
	}
}

// authenticate connects, authenticates with password and quits.
func (a *AuthCheck) authenticate(ctx context.Context, password []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	c, err := dialTLS(ctx, a.opts.Addr, a.host, a.opts.RootCAs)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if err := c.Hello(a.opts.HelloName); err != nil {
		return replyError("EHLO", err)
	}
	if err := c.Auth(smtp.PlainAuth("", a.opts.Username, string(password), a.host)); err != nil {
		return replyError("AUTH", err)
	}
	return replyError("QUIT", c.Quit())
}
