// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"
)

// ErrInvalidLoginToken means a login token is not as NewLoginToken writes
// it. Its message never quotes the token.
var ErrInvalidLoginToken = errors.New("invalid login token")

// LoginLinkLifetime is how long a login link may be used once sent (#59).
const LoginLinkLifetime = 15 * time.Minute

// LoginRequestRetention is how long a login request is kept after its link
// expired or was used, then deleted (ADR 0006).
const LoginRequestRetention = 24 * time.Hour

// LoginTokenLength is the length of a login token: 43 base62 characters,
// about 256 random bits, safe in a URL fragment.
const LoginTokenLength = 43

// loginDigestContext separates the digests of login tokens from those of
// API keys, made with the same secret.
const loginDigestContext = "login-token:"

// LoginToken is the secret of a login link: whoever holds it may obtain an
// API key for the address it was sent to, once. It is a secret: never log
// it. The zero value is not a valid token.
type LoginToken struct {
	value string
}

// NewLoginToken draws a new token from random, such as crypto/rand.Reader.
func NewLoginToken(random io.Reader) (LoginToken, error) {
	s, err := randomBase62(random, LoginTokenLength)
	if err != nil {
		return LoginToken{}, fmt.Errorf("draw login token: %w", err)
	}
	return LoginToken{value: s}, nil
}

// ParseLoginToken reads a token as NewLoginToken writes it, exactly.
func ParseLoginToken(s string) (LoginToken, error) {
	if len(s) != LoginTokenLength || !isBase62(s) {
		return LoginToken{}, ErrInvalidLoginToken
	}
	return LoginToken{value: s}, nil
}

// String returns the whole token: a secret, sent once to the address.
func (t LoginToken) String() string {
	return t.value
}

// TokenDigest is the HMAC-SHA256 of a login token: what is stored in place
// of the token.
type TokenDigest [sha256.Size]byte

// DigestLoginToken returns the TokenDigest of t under secret.
func DigestLoginToken(secret []byte, t LoginToken) TokenDigest {
	return hmacSHA256(secret, loginDigestContext+t.value)
}

// LoginRequest is what is kept of a login link sent to an address: the
// digest of its token, never the token itself.
type LoginRequest struct {
	// Digest is the TokenDigest of the link's token.
	Digest TokenDigest
	// Email is the address the link was sent to, and the account the key
	// it gives is issued to.
	Email Email
	// CreatedAt is when the link was sent.
	CreatedAt time.Time
	// ExpiresAt is when the link stops being valid.
	ExpiresAt time.Time
}

// NewLoginRequest returns the request of a link of digest d sent to to at
// at, valid for LoginLinkLifetime.
func NewLoginRequest(d TokenDigest, to Email, at time.Time) LoginRequest {
	at = at.UTC()
	return LoginRequest{Digest: d, Email: to, CreatedAt: at, ExpiresAt: at.Add(LoginLinkLifetime)}
}
