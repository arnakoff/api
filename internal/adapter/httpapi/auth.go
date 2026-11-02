// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
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

// keyGate admits to the check routes the requests carrying a valid API key
// in an Authorization field of the Bearer scheme (RFC 6750, section 2.1),
// within the quota of their key, known by its identifier. A key that
// cannot be checked fails closed with a 503: no request passes unchecked.
// The key is never logged.
//
// Each client address, known by clientOf, also has a budget of refused
// keys, spent before its key is checked and given back unless the key is
// refused: a flood of well-formed but unknown keys, which anyone can make,
// reaches the key store at a bounded pace per address. Its RateLimit
// fields are only set on its 429. A budget too full to track one more
// address lets that address be checked: a flood from many addresses must
// not lock every new client out.
type keyGate struct {
	auth     Authenticator
	logger   *slog.Logger
	ps       problems
	perKey   *quota
	refusals *quota
	clientOf func(*http.Request) string
}

// require answers next only for a request g admits.
func (g *keyGate) require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header)
		if !ok {
			w.Header().Set("WWW-Authenticate", bearerChallenge)
			g.ps.write(w, r, missingAPIKey)
			return
		}
		client := g.clientOf(r)
		if d := g.refusals.limiter.Allow(client); !d.Allowed && !d.Full {
			g.refusals.deny(w, r, d)
			return
		}
		rec, err := g.authenticate(r.Context(), token)
		if !refused(err) {
			g.refusals.limiter.Refund(client)
		}
		switch {
		case err == nil:
			if g.perKey.admit(w, r, strconv.FormatInt(rec.ID, 10)) {
				next.ServeHTTP(w, r)
			}
		case refused(err):
			w.Header().Set("WWW-Authenticate", invalidChallenge)
			g.ps.write(w, r, invalidAPIKey)
		default:
			g.logger.ErrorContext(r.Context(), "API key not checked", "error", err.Error())
			g.ps.write(w, r, authenticationUnavailable)
		}
	})
}

// authenticate returns the record of the key token; an empty token, sent
// as several Authorization fields, is invalid.
func (g *keyGate) authenticate(ctx context.Context, token string) (domain.KeyRecord, error) {
	if token == "" {
		return domain.KeyRecord{}, domain.ErrInvalidAPIKey
	}
	return g.auth.Authenticate(ctx, token)
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
