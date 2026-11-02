// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/arnakoff/api/internal/platform/logging"
)

// instancePrefix makes the instance of a problem from the request
// identifier: a tag URI (RFC 4151), which names without locating.
const instancePrefix = "tag:arnakoff.com,2026:request:"

// problem is a type of error the API answers. Every one has a Problem…
// example in the contract, which arnakoff/web documents at the type URI.
type problem struct {
	status int
	code   string
	title  string
	detail string
}

// problemBody is an RFC 9457 problem detail, as the contract's Problem
// schema describes it.
type problemBody struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
	Code     string `json:"code"`
}

// problems writes the problems of the API, typed under base.
type problems struct {
	base string
}

// problemsOf returns the problem writer opts configures.
func problemsOf(opts Options) problems {
	return problems{base: opts.ProblemBase}
}

// write sends p, the answer to r, as application/problem+json. Its
// instance names the request.
func (ps problems) write(w http.ResponseWriter, r *http.Request, p problem) {
	body := ps.body(p)
	if id, ok := logging.RequestID(r.Context()); ok {
		body.Instance = instancePrefix + id
	}
	writeJSON(w, "application/problem+json", p.status, body)
}

// writeJSON sends body, of the JSON media type contentType, with status.
func writeJSON(w http.ResponseWriter, contentType string, status int, body any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	// The status is sent: an encoding error can only be a broken connection.
	_ = json.NewEncoder(w).Encode(body)
}

// body returns the body of p, without instance: its type is the page of
// its code under the base.
func (ps problems) body(p problem) problemBody {
	return problemBody{
		Type: ps.base + p.code, Title: p.title, Status: p.status, Detail: p.detail, Code: p.code,
	}
}
