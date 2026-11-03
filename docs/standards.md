# Standards et versions

Référentiel contraignant : chaque règle cite sa norme. Les versions ont été vérifiées le 29/10/2026 sur les sources indiquées ; elles se revérifient avec le skill `check-versions`, jamais de mémoire.

## Versions vérifiées

| Outil | Version | Source |
|---|---|---|
| Go | 1.27.3 (`go 1.27.0` et `toolchain go1.27.3` dans `go.mod`) | `go.dev/dl/?mode=json` |
| Image de build | `golang:1.27.3-alpine3.24`, épinglée par le digest de son index multiplateforme (`sha256:d200d2dd…3deb`), vérifiée le 03/11/2026 | Docker Hub `library/golang`, `docker buildx imagetools inspect` |
| Image d'exécution | `gcr.io/distroless/static-debian13:nonroot`, épinglée par le digest de son index multiplateforme (`sha256:25ac8f0c…8329`), vérifiée le 03/11/2026 | `gcr.io`, `docker buildx imagetools inspect` |
| golangci-lint | v2.15.0 (configuration au format `version: "2"`) | GitHub releases |
| gofumpt | v0.12.0 | GitHub releases |
| gosec | v2.30.0 (via golangci-lint) | GitHub releases |
| govulncheck (`golang.org/x/vuln`) | v1.9.0 | `proxy.golang.org` |
| SDK MCP Go (`modelcontextprotocol/go-sdk`, MIT en transition vers Apache-2.0 d'après son `LICENSE`, les deux compatibles ; paquets liés à `cmd/api` et `cmd/mcp` : `google/jsonschema-go` et `segmentio/asm`, `segmentio/encoding` (MIT), `yosida95/uritemplate/v3` (BSD-3-Clause), `golang.org/x/oauth2` (BSD-3-Clause) ; `golang-jwt/jwt` n'est pas lié) | v1.8.0 (publiée le 04/09/2026), revérifiée le 03/11/2026 | `proxy.golang.org` |
| JSON Schema des outils MCP (`google/jsonschema-go`, MIT) : le type de schéma du SDK MCP, importé pour décrire l'entrée de l'outil `check` ; même version que celle requise par le SDK | v0.4.3 (publiée le 17/04/2026, dernier commit le 22/05/2026), vérifiée le 03/11/2026 | `proxy.golang.org`, GitHub |
| oapi-codegen | v2.8.1 | `proxy.golang.org` |
| benchstat (`golang.org/x/perf`, sans tag : pseudo-version) | v0.0.0-20261009192801-be2c69fb417e | `proxy.golang.org` |
| vegeta (`tsenart/vegeta/v12`, tests de charge, licence MIT) | v12.13.0, vérifiée le 02/11/2026 | GitHub releases, `proxy.golang.org` |
| `getkin/kin-openapi` (dépendance d'oapi-codegen, et validation du contrat dans les tests : `api` et `internal/adapter/contracttest`) ; oapi-codegen v2.8.1 sert aussi de bibliothèque dans le test de génération de `api` | v0.152.0, relevée au-dessus de la v0.142.0 tirée par oapi-codegen (GO-2026-6095, GO-2026-6112, corrigées en v0.144.0) | `proxy.golang.org` |
| Numéros de téléphone (`nyaruka/phonenumbers`) | v1.8.1 | `proxy.golang.org` |
| IBAN (`jacoelho/banking`) | v1.13.0 | `proxy.golang.org` |
| JSON Schema (`santhosh-tekuri/jsonschema/v6`) | v6.0.3 | `proxy.golang.org` |
| Noms de domaine (`golang.org/x/net` : `idna`, `publicsuffix`) | v0.62.0 (publiée le 27/10/2026, retenue malgré le délai de carence : la v0.61.0 est touchée par GO-2026-6860 et GO-2026-6866, dans `x/net/html`) | `proxy.golang.org`, `vuln.go.dev` |
| Dédoublonnage des appels (`golang.org/x/sync` : `singleflight`, BSD-3-Clause) | v0.24.0 (publiée le 23/09/2026) | `proxy.golang.org` |
| Caractères confusables (`go.mau.fi/util` : `confusable`, MPL-2.0, squelette UTS #39 généré depuis `confusables.txt` d'Unicode 17.0.0) | v0.11.0 (publiée le 16/10/2026) ; seul le paquet `confusable` est compilé, il n'importe que `golang.org/x/text` | `proxy.golang.org` |
| Limitation du débit sortant (`golang.org/x/time` : `rate`, BSD-3-Clause) | v0.16.0 (publiée le 19/08/2026) | `proxy.golang.org` |
| PostgreSQL, pilote `jackc/pgx/v5` (MIT ; dépendances `jackc/puddle/v2`, `pgpassfile`, `pgservicefile`, `pgerrcode`, toutes MIT) | v5.11.0 (publiée le 07/09/2026) ; la v5.12.0 (31/10/2026) attend la fin du délai de carence de 7 jours | `proxy.golang.org` |
| Object Storage S3 (`aws/aws-sdk-go-v2/service/s3`, Apache-2.0 ; dépendances `aws/aws-sdk-go-v2` et ses modules internes, `aws/smithy-go`, toutes Apache-2.0 et du même éditeur, sans autre module tiers) : instantanés dans un bucket Scaleway fr-par (ADR 0005, #236). Préféré à `minio/minio-go/v7` (Apache-2.0), qui tire une douzaine de modules tiers. Requêtes par le client HTTP sortant de `platform` (méthode `Do`), sans lecture de l'environnement `AWS_*` | `service/s3` v1.114.5 (publiée le 22/10/2026), `aws-sdk-go-v2` v1.47.4, `smithy-go` v1.28.6 ; la v1.114.6 (27/10/2026) attend la fin du délai de carence de 7 jours ; vérifiées le 03/11/2026 | `proxy.golang.org`, `LICENSE.txt` du module |
| Migrations SQL (`golang-migrate/migrate/v4`, MIT : paquets `database/pgx/v5` et `source/iofs` seulement) | v4.20.1 (publiée le 09/09/2026), choisie par le propriétaire le 02/11/2026 (ADR 0004) | `proxy.golang.org` |
| Serveur PostgreSQL de la CI (conteneur de service du job Test) | `postgres:17.11` (publiée le 01/11/2026), épinglée par digest : la dernière version majeure de Managed Database for PostgreSQL de Scaleway en fr-par (ADR 0005, relevé du 03/11/2026) | Docker Hub `library/postgres` |
| `actions/checkout` / `actions/setup-go` | v7.0.1 / v7.0.0, épinglées par SHA | GitHub releases |
| `golangci/golangci-lint-action` | v9.3.0, épinglée par SHA | GitHub releases |
| `github/codeql-action` | v4.39.1, épinglée par SHA | GitHub tags |
| `docker/build-push-action` | v7.4.0, épinglée par SHA | GitHub releases |
| `docker/setup-buildx-action` / `docker/login-action` / `docker/metadata-action` | v4.4.1 / v4.7.0 / v6.2.0, épinglées par SHA, vérifiées le 03/11/2026 | GitHub releases |
| `sigstore/cosign-installer` | v4.1.2, épinglée par SHA, vérifiée le 03/11/2026 ; installe cosign v3.1.4 (`cosign-release`) après vérification par sa version d'amorçage | GitHub releases |
| `googleapis/release-please-action` | v5.0.0 (embarque release-please 17.6.0, d’après son `package-lock.json` et le journal du workflow), épinglée par SHA | GitHub releases |
| `actions/create-github-app-token` | v3.3.0, épinglée par SHA | GitHub releases |
| `ossf/scorecard-action` | v2.5.0, épinglée par SHA | GitHub releases |
| cosign / syft | v3.1.4 / v1.55.0 | GitHub releases |
| OpenAPI | 3.2.1 pour `api/openapi.yaml`, dernière version publiée (10/09/2026), décision du propriétaire du 02/11/2026. kin-openapi v0.152.0 la valide (prise en charge partielle des nouveautés de la 3.2, que le contrat n'utilise pas) ; oapi-codegen v2.8.1 la traite comme une 3.1 ou plus (`IsOpenAPI31OrLater`), et un test génère le code depuis le contrat. `@readme/openapi-parser`, sous starlight-openapi 0.27.0, refuse la 3.2 en validation (« OpenAPI 3.2 is currently unsupported. ») mais starlight-openapi n'appelle que `bundle` : rendu **à vérifier** dans `arnakoff/web` | `OAI/OpenAPI-Specification` releases, sources des trois outils |

Outils de développement épinglés avec la directive `tool` de `go.mod` (`go tool govulncheck`, `go tool gofumpt`, `go tool benchstat`, `go tool oapi-codegen`, `go tool vegeta`). vegeta est épinglé ainsi plutôt qu'installé à part : sa v12.13.0 fixe `golang.org/x/net` v0.27.0 (GO-2026-6603, GO-2026-6610 à 6612, GO-2026-6617, corrigées en v0.60.0 ; GO-2026-5026, en v0.55.0), que la directive `tool` relève à la version du module ; ses dépendances (client Prometheus, Apache-2.0 ; les autres MIT ou BSD) ne sont liées à aucun binaire livré. Leurs dépendances se vérifient avec `go tool govulncheck tool`. golangci-lint s'installe en binaire à la version ci-dessus, comme le recommande son projet.

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
- Le contrat est validé par `go test` (kin-openapi, exemples compris), donc en CI ; les tests des handlers vérifient leurs réponses réelles avec `internal/adapter/contracttest`, qui y refuse tout champ non documenté. Un client, lui, ignore les champs qu'il ne connaît pas.
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
- Tags de version : un ruleset de tags (« version tags ») sur `refs/tags/v*` interdit leur création, modification et suppression, sauf à l'App GitHub `arnakoff-release` de release-please. Seule une vraie version déclenche donc la publication et la signature de l'image (#233).
- CodeQL (Go), [OpenSSF Scorecard](https://github.com/ossf/scorecard), analyse des secrets et protection au push activées.
- Image : build multi-étapes, exécution sur `gcr.io/distroless/static-debian13:nonroot` (Debian 13, tag vérifié le 29/10/2026) épinglée par digest, utilisateur non root, système de fichiers en lecture seule. Publiée sur `ghcr.io/arnakoff/api` à chaque tag de version par `.github/workflows/image.yml` (#69) : `linux/amd64` et `linux/arm64`, SBOM (scanner Syft de BuildKit) et provenance `mode=max` attachés par `docker/build-push-action`, signature sans clé (cosign, jeton OIDC du job) de l'index par son digest, sans cache de build. Attestation de provenance [SLSA](https://slsa.dev/) niveau 3 visé.
