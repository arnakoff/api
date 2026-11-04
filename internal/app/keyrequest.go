// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// ErrLoginLinkInvalid means a login token is malformed, unknown, expired or
// already used: the four are told apart to no one.
var ErrLoginLinkInvalid = errors.New("login link invalid, expired or used")

// SelfServiceKeyLabel is the label of a key issued by a login link: its
// record names the account by its address.
const SelfServiceKeyLabel = "libre-service"

// The mail of a login link, as the owner approved it (#59).
const (
	loginSubject = "Votre lien pour obtenir une clé API Arnakoff"
	loginBody    = "Bonjour,\n\n" +
		"Pour obtenir votre clé API Arnakoff, ouvrez ce lien dans les %d minutes :\n" +
		"%s\n\n" +
		"Ce lien ne sert qu'une fois. Si vous n'avez rien demandé, ignorez ce message : aucune clé ne sera créée.\n\n" +
		"Arnakoff — https://arnakoff.com\n"
)

// LoginRequestStore keeps the login requests.
type LoginRequestStore interface {
	// CreateLoginRequest stores r.
	CreateLoginRequest(ctx context.Context, r domain.LoginRequest) error
	// UseLoginRequest marks used at at the request of digest d, unused and
	// unexpired at at, and returns its address; otherwise it returns
	// ErrLoginLinkInvalid. Two calls never both succeed.
	UseLoginRequest(ctx context.Context, d domain.TokenDigest, at time.Time) (domain.Email, error)
}

// AccountKeyIssuer issues the key of an account.
type AccountKeyIssuer interface {
	// ReplaceAccountKey revokes at r.CreatedAt every active key of the
	// account r.Email.Account, stores r and returns it with its identifier
	// set, all at once.
	ReplaceAccountKey(ctx context.Context, r domain.KeyRecord) (domain.KeyRecord, error)
}

// Mailer sends a plain text mail.
type Mailer interface {
	// Send sends to to the mail of subject and body. Its errors never
	// quote the address.
	Send(ctx context.Context, to domain.Email, subject, body string) error
}

// KeyRequestDeps are what KeyRequests needs.
type KeyRequestDeps struct {
	Logins LoginRequestStore
	Keys   AccountKeyIssuer
	Mailer Mailer
	// Secret is the HMAC key of the API keys, which digests the login
	// tokens too.
	Secret []byte
	// Environment is that of the keys issued.
	Environment domain.KeyEnvironment
	Clock       Clock
	// Random is the source of the tokens and keys, such as
	// crypto/rand.Reader.
	Random io.Reader
	// LinkBase is the page of the site a login link opens, the token in
	// its fragment: the fragment never reaches a server's log.
	LinkBase string
}

// KeyRequests issues free API keys on request: a login link mailed to an
// address, then a key to whoever opens it in time (#59).
type KeyRequests struct {
	deps KeyRequestDeps
}

// NewKeyRequests returns the KeyRequests of deps.
func NewKeyRequests(deps KeyRequestDeps) *KeyRequests {
	deps.Secret = slices.Clone(deps.Secret)
	return &KeyRequests{deps: deps}
}

// RequestLink mails to to a login link valid for domain.LoginLinkLifetime
// and stores the digest of its token. Its errors never quote the address.
func (k *KeyRequests) RequestLink(ctx context.Context, to domain.Email) error {
	tok, err := domain.NewLoginToken(k.deps.Random)
	if err != nil {
		return err
	}
	r := domain.NewLoginRequest(domain.DigestLoginToken(k.deps.Secret, tok), to, k.deps.Clock.Now())
	if err := k.deps.Logins.CreateLoginRequest(ctx, r); err != nil {
		return fmt.Errorf("store login request for %s: %w", to.Redacted(), err)
	}
	link := k.deps.LinkBase + "#" + tok.String()
	if err := k.deps.Mailer.Send(ctx, to, loginSubject, fmt.Sprintf(loginBody, int(domain.LoginLinkLifetime/time.Minute), link)); err != nil {
		return fmt.Errorf("mail login link to %s: %w", to.Redacted(), err)
	}
	return nil
}

// Redeem uses the login link of token raw and issues a key to its account,
// revoking the key the account had. The key is shown once and never kept.
// It fails with ErrLoginLinkInvalid for a link that cannot be used.
func (k *KeyRequests) Redeem(ctx context.Context, raw string) (domain.APIKey, domain.KeyRecord, error) {
	tok, err := domain.ParseLoginToken(raw)
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, ErrLoginLinkInvalid
	}
	now := k.deps.Clock.Now().UTC()
	to, err := k.deps.Logins.UseLoginRequest(ctx, domain.DigestLoginToken(k.deps.Secret, tok), now)
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, fmt.Errorf("use login request: %w", err)
	}
	key, err := domain.NewAPIKey(k.deps.Environment, k.deps.Random)
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, err
	}
	rec, err := k.deps.Keys.ReplaceAccountKey(ctx, domain.KeyRecord{
		Digest: domain.DigestAPIKey(k.deps.Secret, key), Environment: k.deps.Environment,
		Label: SelfServiceKeyLabel, Hint: key.Hint(), Email: to, CreatedAt: now,
	})
	if err != nil {
		return domain.APIKey{}, domain.KeyRecord{}, fmt.Errorf("issue API key to %s: %w", to.Redacted(), err)
	}
	return key, rec, nil
}
