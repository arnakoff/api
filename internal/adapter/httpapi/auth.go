// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
)

// Authenticator tells whether an API key may call the API.
type Authenticator interface {
	// Authenticate returns the record of the key raw. A malformed,
	// unknown or revoked key is refused with domain.ErrInvalidAPIKey,
	// app.ErrKeyNotFound or app.ErrKeyRevoked; any other error means the
	// key could not be checked.
	Authenticate(ctx context.Context, raw string) (domain.KeyRecord, error)
}

// Challenges of the WWW-Authenticate field (RFC 6750, section 3): without
// an error code when the request carries no key, invalid_token when its
// key is refused.
const (
	bearerChallenge  = `Bearer realm="arnakoff"` //nolint:gosec // G101: an authentication challenge, no credential.
	invalidChallenge = bearerChallenge + `, error="invalid_token"`
)

// requireKey answers next only for a request carrying a valid API key in
// an Authorization field of the Bearer scheme (RFC 6750, section 2.1). A
// key that cannot be checked fails closed with a 503: no request passes
// unchecked. The key is never logged.
func requireKey(auth Authenticator, logger *slog.Logger, ps problems, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header)
		if !ok {
			w.Header().Set("WWW-Authenticate", bearerChallenge)
			ps.write(w, r, missingAPIKey)
			return
		}
		var err error
		if token == "" {
			err = domain.ErrInvalidAPIKey
		} else {
			_, err = auth.Authenticate(r.Context(), token)
		}
		switch {
		case err == nil:
			next.ServeHTTP(w, r)
		case refused(err):
			w.Header().Set("WWW-Authenticate", invalidChallenge)
			ps.write(w, r, invalidAPIKey)
		default:
			logger.ErrorContext(r.Context(), "API key not checked", "error", err.Error())
			ps.write(w, r, authenticationUnavailable)
		}
	})
}

// bearerToken returns the token of the Authorization field of h, and false
// when h has no such field of the Bearer scheme, whose name is matched in
// any case (RFC 9110, section 11.1). Several Authorization fields give an
// empty token, which no key matches.
func bearerToken(h http.Header) (string, bool) {
	fields := h.Values("Authorization")
	if len(fields) == 0 {
		return "", false
	}
	scheme, token, _ := strings.Cut(fields[0], " ")
	if !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	if len(fields) > 1 {
		return "", true
	}
	return token, true
}

// errNoAuthenticator is the error of noAuthenticator.
var errNoAuthenticator = errors.New("no authenticator configured")

// noAuthenticator stands for an unset Options.Auth: it checks no key, so
// every request needing one fails closed.
type noAuthenticator struct{}

func (noAuthenticator) Authenticate(context.Context, string) (domain.KeyRecord, error) {
	return domain.KeyRecord{}, errNoAuthenticator
}

// refused reports whether err refuses the key, as opposed to failing to
// check it.
func refused(err error) bool {
	return errors.Is(err, domain.ErrInvalidAPIKey) ||
		errors.Is(err, app.ErrKeyNotFound) ||
		errors.Is(err, app.ErrKeyRevoked)
}
