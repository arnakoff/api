// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"
	"net/url"

	"github.com/arnakoff/api/internal/adapter/wire"
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
		writeJSON(w, "application/json", http.StatusOK, wire.ReportOf(in.Kind(), c.Check(r.Context(), in)))
	})
}

// checkInput returns the input the query asks to check, or the problem
// refusing it.
func checkInput(query url.Values) (domain.Input, problem, bool) {
	return parseInput(query.Get("q"), query.Get("type"), query.Has("type"))
}

// parseInput returns the input raw names, of the kind typ names when
// hasType, or the problem refusing it: raw is required, typ names a kind.
func parseInput(raw, typ string, hasType bool) (domain.Input, problem, bool) {
	if raw == "" {
		return nil, missingParameter, false
	}
	kind := domain.KindAuto
	if hasType {
		k, err := domain.ParseKind(typ)
		if err != nil {
			return nil, problemFor(err), false
		}
		kind = k
	}
	in, err := domain.ParseInput(raw, kind)
	if err != nil {
		return nil, inputProblem(kind, err), false
	}
	return in, problem{}, true
}
