// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"

	"github.com/arnakoff/api/internal/platform/logging"
)

// requestIDHeader carries the request identifier, both ways.
const requestIDHeader = "X-Request-ID"

// maxRequestIDLen bounds a request identifier sent by a client.
const maxRequestIDLen = 64

// withRequestID gives every request an identifier: the client's when it is
// bounded, a new one from newID otherwise. The identifier is sent back and
// added to every line logged with the request's context.
func withRequestID(newID func() string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID(id) {
			id = newID()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(logging.WithRequestID(r.Context(), id)))
	})
}

// validRequestID reports whether id is 1 to maxRequestIDLen ASCII letters,
// digits, ".", "_" or "-": safe in a log line and in a header.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, c := range []byte(id) {
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}
