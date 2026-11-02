// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/arnakoff/api/internal/domain"
)

// check answers GET /v1/check: the report of the input q, of the kind type
// when it is given, detected otherwise. The report keeps the no-store
// default of secure: its input may be personal data.
func check(c Checker, ps problems) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in, p, ok := checkInput(r.URL.Query())
		if !ok {
			ps.write(w, r, p)
			return
		}
		body := reportOf(in.Kind(), c.Check(r.Context(), in))
		w.Header().Set("Content-Type", "application/json")
		// The status is sent: an encoding error can only be a broken connection.
		_ = json.NewEncoder(w).Encode(body)
	})
}

// checkInput returns the input the query asks to check, or the problem
// refusing it: q is required, type, when present, names a kind.
func checkInput(query url.Values) (domain.Input, problem, bool) {
	raw := query.Get("q")
	if raw == "" {
		return nil, missingParameter, false
	}
	kind := domain.KindAuto
	if query.Has("type") {
		k, err := domain.ParseKind(query.Get("type"))
		if err != nil {
			return nil, problemFor(err), false
		}
		kind = k
	}
	in, err := domain.ParseInput(raw, kind)
	if err != nil {
		return nil, problemFor(err), false
	}
	return in, problem{}, true
}
