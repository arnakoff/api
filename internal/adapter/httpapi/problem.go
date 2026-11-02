// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"encoding/json"
	"net/http"
)

// problem is an RFC 9457 problem detail, as the contract's Problem schema
// describes it.
type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

// newProblem returns the problem code, documented at /problemes/<code>.
func newProblem(status int, code, title, detail string) problem {
	return problem{Type: "/problemes/" + code, Title: title, Status: status, Detail: detail, Code: code}
}

// Problems the middleware answers.
var (
	notFound = newProblem(http.StatusNotFound, "not_found",
		"Ressource introuvable", "Aucune ressource ne correspond à ce chemin.")
	methodNotAllowed = newProblem(http.StatusMethodNotAllowed, "method_not_allowed",
		"Méthode non autorisée", "Cette ressource n'accepte pas cette méthode.")
	internalError = newProblem(http.StatusInternalServerError, "internal_error",
		"Erreur interne", "Le serveur a rencontré une erreur. Réessaie plus tard.")
)

// writeProblem sends p as application/problem+json.
func writeProblem(w http.ResponseWriter, p problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	// The status is sent: an encoding error can only be a broken connection.
	_ = json.NewEncoder(w).Encode(p)
}
