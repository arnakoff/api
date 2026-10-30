// SPDX-License-Identifier: AGPL-3.0-only

package httpclienttest

import "net/http"

// Stall is a handler that never answers: it waits for the request to end,
// then aborts the connection. Returning instead would send an empty 200,
// which the client can still read while it closes the connection on its
// own timeout (net/http Transport keeps a response that races the close).
func Stall(_ http.ResponseWriter, r *http.Request) {
	<-r.Context().Done()
	panic(http.ErrAbortHandler) // The only way to end a handler without a response.
}
