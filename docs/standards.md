# Standards et versions

Référentiel contraignant : chaque règle cite sa norme. Les versions ont été vérifiées le 29/10/2026 sur les sources indiquées ; elles se revérifient avec le skill `check-versions`, jamais de mémoire.

## Versions vérifiées

| Outil | Version | Source |
|---|---|---|
| Go | 1.27.3 (`go 1.27.0` et `toolchain go1.27.3` dans `go.mod`) | `go.dev/dl/?mode=json` |
| Image de build | `golang:1.27.3-alpine3.24`, épinglée par digest | Docker Hub `library/golang` |
| golangci-lint | v2.15.0 (configuration au format `version: "2"`) | GitHub releases |
| gofumpt | v0.12.0 | GitHub releases |
| gosec | v2.30.0 (via golangci-lint) | GitHub releases |
| govulncheck (`golang.org/x/vuln`) | v1.9.0 | `proxy.golang.org` |
| SDK MCP Go | v1.8.0 | `proxy.golang.org` |
| oapi-codegen | v2.8.1 | `proxy.golang.org` |
| benchstat (`golang.org/x/perf`, sans tag : pseudo-version) | v0.0.0-20261009192801-be2c69fb417e | `proxy.golang.org` |
| `getkin/kin-openapi` (dépendance d'oapi-codegen) | v0.152.0, relevée au-dessus de la v0.142.0 tirée par oapi-codegen (GO-2026-6095, GO-2026-6112, corrigées en v0.144.0) | `proxy.golang.org` |
| Numéros de téléphone (`nyaruka/phonenumbers`) | v1.8.1 | `proxy.golang.org` |
| IBAN (`jacoelho/banking`) | v1.13.0 | `proxy.golang.org` |
| JSON Schema (`santhosh-tekuri/jsonschema/v6`) | v6.0.3 | `proxy.golang.org` |
| Noms de domaine (`golang.org/x/net` : `idna`, `publicsuffix`) | v0.62.0 (publiée le 27/10/2026, retenue malgré le délai de carence : la v0.61.0 est touchée par GO-2026-6860 et GO-2026-6866, dans `x/net/html`) | `proxy.golang.org`, `vuln.go.dev` |
| Dédoublonnage des appels (`golang.org/x/sync` : `singleflight`, BSD-3-Clause) | v0.24.0 (publiée le 23/09/2026) | `proxy.golang.org` |
| `actions/checkout` / `actions/setup-go` | v7.0.1 / v7.0.0, épinglées par SHA | GitHub releases |
| `golangci/golangci-lint-action` | v9.3.0, épinglée par SHA | GitHub releases |
| `github/codeql-action` | v4.39.1, épinglée par SHA | GitHub tags |
| `docker/build-push-action` | v7.4.0, épinglée par SHA | GitHub releases |
| `googleapis/release-please-action` | v5.0.0 (embarque release-please 17.6.0, d’après son `package-lock.json` et le journal du workflow), épinglée par SHA | GitHub releases |
| `actions/create-github-app-token` | v3.3.0, épinglée par SHA | GitHub releases |
| `ossf/scorecard-action` | v2.5.0, épinglée par SHA | GitHub releases |
| cosign / syft | v3.1.4 / v1.55.0 | GitHub releases |
| OpenAPI | 3.2.1 publiée ; **à vérifier** : prise en charge par oapi-codegen, sinon 3.1.1 | `OAI/OpenAPI-Specification` |

Outils de développement épinglés avec la directive `tool` de `go.mod` (`go tool govulncheck`, `go tool gofumpt`, `go tool benchstat`, `go tool oapi-codegen`). Leurs dépendances se vérifient avec `go tool govulncheck tool`. golangci-lint s'installe en binaire à la version ci-dessus, comme le recommande son projet.

## Go

- [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), [Google Go Style Guide](https://google.github.io/styleguide/go/) (guide, décisions, bonnes pratiques).
- [Go Doc Comments](https://go.dev/doc/comment) : tout identifiant exporté est documenté, phrase complète commençant par son nom.
- Formatage `gofumpt` et imports `goimports` (formateurs de golangci-lint), appliqués en CI.
- Module `github.com/arnakoff/api`, disposition `cmd/` et `internal/` ([Organizing a Go module](https://go.dev/doc/modules/layout)).
- Journalisation `log/slog` en JSON ; routage `net/http.ServeMux` (motifs méthode et chemin) avant toute bibliothèque de routage.

## API HTTP

- [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110) (sémantique HTTP) : méthodes, codes de statut, cache.
- [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) : toutes les erreurs en `application/problem+json` (`type`, `title`, `status`, `detail`, `instance`, extension `code`).
- [RFC 9745](https://www.rfc-editor.org/rfc/rfc9745) (`Deprecation`) et [RFC 8594](https://www.rfc-editor.org/rfc/rfc8594) (`Sunset`) pour retirer une version ou un champ du contrat public, avec un préavis de six mois minimum entre la date de `Deprecation` et celle de `Sunset`.
- En-têtes de quota : `RateLimit-Policy` et `RateLimit` selon [draft-ietf-httpapi-ratelimit-headers-11](https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/) (brouillon, pas encore une RFC : suivre ses révisions), `Retry-After` sur `429`.
- Contrat OpenAPI dans `api/openapi.yaml`, versionné par chemin (`/v1`). Champs JSON en `snake_case`, dates en [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339).
- [OWASP API Security Top 10 (2023)](https://owasp.org/API-Security/editions/2023/en/0x00-header/) comme grille de revue.

## Git et GitHub

- [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) pour les commits et les titres de PR (le titre devient le commit en squash merge) : `feat(signal): flag domains registered less than 30 days ago`.
- [SemVer 2.0.0](https://semver.org/) ; versions et `CHANGELOG.md` ([Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/)) produits par release-please à partir des commits.
- Règles de la branche `main` (ruleset) : PR obligatoire, CI verte, historique linéaire, pas de force-push, résolution des conversations.
- Fichiers communautaires : `README.md`, `LICENSE` (AGPL-3.0-only, texte officiel de gnu.org, ADR 0003), en-tête `SPDX-License-Identifier` dans chaque fichier source, accord de contribution (CLA), `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` ([Contributor Covenant 3.0](https://www.contributor-covenant.org/version/3/0/)), `SECURITY.md` (signalement privé des vulnérabilités GitHub), `SUPPORT.md`, `.github/CODEOWNERS`, formulaires d'issues YAML, modèle de PR, `.editorconfig`, `.gitattributes`.
- Labels : `type:*` (feature, bug, chore, docs, decision), `area:*` (une par couche ou zone), `owner-only` (action réservée au propriétaire : compte, paiement, juridique, DNS), `security`, `source`, et les labels GitHub `good first issue` et `help wanted`.
- Jalons communs aux dépôts : `M0 — Fondations`, `M1 — Moteur de signaux`, `M2 — API publique v1`, `M3 — Production`, `M4 — Lancement`, `M5 — Monétisation`. Épics `[domaine] ÉPIC — …` avec sous-issues GitHub ; une sous-issue commence par `Partie de #E.` et ses dépendances s'écrivent `Bloquée par #N` dans sa section « Liens ».

## Chaîne d'approvisionnement et CI

- Actions GitHub épinglées par SHA complet (commentaire avec la version), `permissions: {}` au niveau du workflow puis le minimum par job, aucun `pull_request_target` sur du code de fork.
- Dependabot pour `gomod`, `github-actions` et `docker`, hebdomadaire, regroupé, avec un délai de carence (`cooldown`) de 7 jours par défaut et de 30 jours pour les versions majeures des modules Go (#90). Le délai ne s'applique qu'aux mises à jour de version, jamais aux mises à jour de sécurité ([référence des options Dependabot](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference), consultée le 29/10/2026).
- CodeQL (Go), [OpenSSF Scorecard](https://github.com/ossf/scorecard), analyse des secrets et protection au push activées.
- Image : build multi-étapes, exécution sur `gcr.io/distroless/static-debian13:nonroot` (Debian 13, tag vérifié le 29/10/2026) épinglée par digest, utilisateur non root, système de fichiers en lecture seule. SBOM (syft), signature sans clé (cosign), attestation de provenance [SLSA](https://slsa.dev/) niveau 3 visé.
