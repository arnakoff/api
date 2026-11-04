// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/arnakoff/api/internal/adapter/wire"
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
		http.StatusUnprocessableEntity, "invalid_input", "Entrée invalide", wire.InvalidInput(domain.KindAuto),
	}
	invalidBody = problem{
		http.StatusBadRequest, "invalid_body", "Corps de requête invalide",
		"Le corps de la requête doit être un objet JSON contenant une liste items d'au moins un élément.",
	}
	invalidDemoBody = problem{
		http.StatusBadRequest, "invalid_body", "Corps de requête invalide",
		"Le corps de la requête doit être un objet JSON contenant q et turnstile_token.",
	}
	invalidKeyRequestBody = problem{
		http.StatusBadRequest, "invalid_body", "Corps de requête invalide",
		"Le corps de la requête doit être un objet JSON contenant email et turnstile_token.",
	}
	invalidKeyRedeemBody = problem{
		http.StatusBadRequest, "invalid_body", "Corps de requête invalide",
		"Le corps de la requête doit être un objet JSON contenant token.",
	}
	invalidLoginLink = problem{
		http.StatusUnprocessableEntity, "invalid_login_link", "Lien de connexion invalide",
		"Le lien de connexion est invalide, expiré ou déjà utilisé. Demande un nouveau lien.",
	}
	keyRequestUnavailable = problem{
		http.StatusServiceUnavailable, "key_request_unavailable", "Demande de clé indisponible",
		"La demande de clé ne peut pas être traitée pour le moment. Réessaie plus tard.",
	}
	missingAPIKey = problem{
		http.StatusUnauthorized, "missing_api_key", "Clé API manquante",
		"Cette ressource demande une clé API, envoyée dans l'en-tête Authorization avec le schéma Bearer.",
	}
	invalidAPIKey = problem{
		http.StatusUnauthorized, "invalid_api_key", "Clé API invalide",
		"La clé API est mal formée, inconnue ou révoquée.",
	}
	authenticationUnavailable = problem{
		http.StatusServiceUnavailable, "authentication_unavailable", "Authentification indisponible",
		"La clé API ne peut pas être vérifiée pour le moment. Réessaie plus tard.",
	}
	challengeFailed = problem{
		http.StatusForbidden, "challenge_failed", "Vérification anti-abus échouée",
		"Le jeton anti-abus est absent, invalide, expiré ou déjà utilisé. Recharge la vérification et réessaie.",
	}
	challengeUnavailable = problem{
		http.StatusServiceUnavailable, "challenge_unavailable", "Vérification anti-abus indisponible",
		"Le jeton anti-abus ne peut pas être vérifié pour le moment. Réessaie plus tard.",
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

// invalidInputOf is the problem answering an input invalid for the kind it
// was forced into, by kind: the detail names the kind. Detected, KindAuto,
// the input was of no kind: the detail names them all.
var invalidInputOf = map[domain.Kind]problem{
	domain.KindAuto:   invalidInput,
	domain.KindDomain: invalidInputFor(domain.KindDomain),
	domain.KindEmail:  invalidInputFor(domain.KindEmail),
	domain.KindPhone:  invalidInputFor(domain.KindPhone),
	domain.KindIBAN:   invalidInputFor(domain.KindIBAN),
	domain.KindSIREN:  invalidInputFor(domain.KindSIREN),
	domain.KindSIRET:  invalidInputFor(domain.KindSIRET),
}

// invalidInputFor returns invalidInput with the detail naming k.
func invalidInputFor(k domain.Kind) problem {
	p := invalidInput
	p.detail = wire.InvalidInput(k)
	return p
}

// tooManyItems is the problem answering a batch of more than limit items.
func tooManyItems(limit int) problem {
	return problem{
		http.StatusUnprocessableEntity, "too_many_items", "Trop d'éléments",
		fmt.Sprintf("Le lot dépasse le nombre maximal d'éléments autorisé : %d.", limit),
	}
}

// problemCatalog lists every problem the API answers but tooManyItems,
// whose detail states the configured limit: a test holds them equal to the
// contract's examples, the list arnakoff/web documents.
var problemCatalog = append([]problem{
	invalidKind, missingParameter, notFound, methodNotAllowed, contentTooLarge,
	unsupportedMediaType, invalidBody, invalidDemoBody, missingAPIKey, invalidAPIKey,
	authenticationUnavailable, challengeFailed, challengeUnavailable, tooManyRequests, internalError,
	serviceUnavailable, invalidKeyRequestBody, invalidKeyRedeemBody, invalidLoginLink, keyRequestUnavailable,
}, slices.Collect(maps.Values(invalidInputOf))...)

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

// inputProblem returns the problem answering err, the error of parsing an
// input of kind k, KindAuto when it was detected.
func inputProblem(k domain.Kind, err error) problem {
	if errors.Is(err, domain.ErrUnrecognisedInput) {
		return invalidInputOf[k]
	}
	return problemFor(err)
}
