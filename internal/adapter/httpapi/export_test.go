// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import "net/http"

// Wrap exposes wrap: it applies the request middleware to any handler.
var Wrap = wrap

// Route exposes route: it dispatches to any mux.
func Route(opts Options, mux *http.ServeMux) http.Handler {
	return route(problemsOf(opts), mux)
}

// Catalog exposes problemCatalog.
var Catalog = problemCatalog

// Answer returns a handler answering p, as a handler of the API does.
func Answer(opts Options, p problem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemsOf(opts).write(w, r, p)
	})
}

// ProblemFor returns the status and the code of the problem answering err.
func ProblemFor(err error) (int, string) {
	p := problemFor(err)
	return p.status, p.code
}
