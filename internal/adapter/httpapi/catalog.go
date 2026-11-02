// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"errors"
	"net/http"

	"github.com/arnakoff/api/internal/domain"
)

// The problems the API answers. A title is the same for every occurrence;
// a detail never quotes the input, which may be personal data.
var (
	invalidKind = problem{
		http.StatusBadRequest, "invalid_parameter", "Paramètre invalide",
		"Le paramètre type doit valoir domain, email, phone, iban, siren ou siret.",
	}
	missingParameter = problem{
		http.StatusBadRequest, "missing_parameter", "Paramètre manquant",
		"Le paramètre q est obligatoire.",
	}
	notFound = problem{
		http.StatusNotFound, "not_found", "Ressource introuvable",
		"Aucune ressource ne correspond à ce chemin.",
	}
	methodNotAllowed = problem{
		http.StatusMethodNotAllowed, "method_not_allowed", "Méthode non autorisée",
		"Cette ressource n'accepte pas cette méthode.",
	}
	contentTooLarge = problem{
		http.StatusRequestEntityTooLarge, "content_too_large", "Corps de requête trop gros",
		"Le corps de la requête dépasse la taille autorisée.",
	}
	unsupportedMediaType = problem{
		http.StatusUnsupportedMediaType, "unsupported_media_type",
		"Type de contenu non pris en charge", "Le corps de la requête doit être en application/json.",
	}
	invalidInput = problem{
		http.StatusUnprocessableEntity, "invalid_input", "Entrée invalide",
		"L'entrée n'est ni un domaine, ni une URL, ni une adresse e-mail, ni un numéro de téléphone, " +
			"ni un IBAN, ni un SIREN, ni un SIRET.",
	}
	tooManyRequests = problem{
		http.StatusTooManyRequests, "too_many_requests", "Trop de requêtes",
		"Le quota de requêtes est atteint. Réessaie après le délai de l'en-tête Retry-After.",
	}
	internalError = problem{
		http.StatusInternalServerError, "internal_error", "Erreur interne",
		"Le serveur a rencontré une erreur. Réessaie plus tard.",
	}
	serviceUnavailable = problem{
		http.StatusServiceUnavailable, "service_unavailable", "Service indisponible",
		"Le service n'est pas encore prêt. Réessaie plus tard.",
	}
)

// problemCatalog lists every problem the API answers: a test holds it equal
// to the contract's examples, the list arnakoff/web documents.
var problemCatalog = []problem{
	invalidKind, missingParameter, notFound, methodNotAllowed, contentTooLarge,
	unsupportedMediaType, invalidInput, tooManyRequests, internalError, serviceUnavailable,
}

// problemFor returns the problem answering err, an error of the domain
// matched with errors.Is; any other error is an internal error.
func problemFor(err error) problem {
	switch {
	case errors.Is(err, domain.ErrUnknownKind):
		return invalidKind
	case errors.Is(err, domain.ErrUnrecognisedInput):
		return invalidInput
	default:
		return internalError
	}
}
