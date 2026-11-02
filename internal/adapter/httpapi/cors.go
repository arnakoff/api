// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import "net/http"

// CORS fields of the demo route (Fetch Standard, section 3.2.3).
const (
	// corsExposed are the response fields the site may read besides the
	// safelisted ones: the quota, its delay and the request identifier a
	// problem names.
	corsExposed = "RateLimit, RateLimit-Policy, Retry-After, " + requestIDHeader
	// corsMaxAge is how long, in seconds, a browser may keep a preflight.
	corsMaxAge = "600"
)

// allowOrigin answers next and lets a browser at origin, and at no other,
// read the answer. No credential is allowed: the demo needs none.
func allowOrigin(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")
		if isOrigin(r, origin) {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Expose-Headers", corsExposed)
		}
		next.ServeHTTP(w, r)
	})
}

// preflight answers the CORS preflight of a POST of JSON from origin with
// a 204 allowing it; any other preflight gets a 204 allowing nothing, which
// the browser takes as a refusal.
func preflight(origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")
		if isOrigin(r, origin) && r.Header.Get("Access-Control-Request-Method") == http.MethodPost {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", http.MethodPost)
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			h.Set("Access-Control-Max-Age", corsMaxAge)
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// isOrigin reports whether r comes from origin, a non-empty origin
// compared byte for byte with the Origin field.
func isOrigin(r *http.Request, origin string) bool {
	return origin != "" && r.Header.Get("Origin") == origin
}
