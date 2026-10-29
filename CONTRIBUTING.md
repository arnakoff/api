# Contribuer à Arnakoff

Merci de ton aide. Ce guide couvre l'API, la synchronisation des sources et le serveur MCP ; pour le site, vois le dépôt [`arnakoff/web`](https://github.com/arnakoff/web).

En participant, tu acceptes le [code de conduite](https://github.com/arnakoff/.github/blob/main/CODE_OF_CONDUCT.md). Une faille de sécurité se signale en privé, selon la [politique de sécurité](https://github.com/arnakoff/.github/blob/main/SECURITY.md).

## Avant de commencer

- Cherche si une issue existe déjà. Sinon, ouvre-en une avec l'un des [formulaires](https://github.com/arnakoff/api/issues/new/choose) : bogue, fonctionnalité ou demande de source.
- Une issue, une branche, une pull request. Pour un changement important, discute-en dans l'issue avant d'écrire du code.
- Les issues et les ADR s'écrivent en français. Le code, les commentaires, les messages de commit et les pull requests s'écrivent en anglais.
- N'écris jamais de donnée personnelle réelle (e-mail, numéro de téléphone, IBAN) dans une issue, un test ou un journal.

## Accord de contribution (CLA)

Avant la fusion de ta première pull request, tu signes l'[accord de contribution](https://gist.github.com/EnO33/bd7ba34d77c8cd46555bfbdc53a1401d) ([ADR 0003](docs/adr/0003-licences.md)). Tu gardes tes droits sur ta contribution et tu accordes au projet une licence non exclusive, qui lui permet de la distribuer sous AGPL-3.0 et, en parallèle, sous une licence commerciale. Seul le texte de l'accord fait foi.

- À l'ouverture de ta pull request, CLA assistant y ajoute un commentaire avec un lien de signature ; tu signes avec ton compte GitHub. Tu peux aussi signer d'avance sur [cla-assistant.io/arnakoff/api](https://cla-assistant.io/arnakoff/api).
- Si tu as écrit la contribution dans le cadre de ton emploi, ton employeur doit approuver l'accord avant que tu le signes.
- Si l'accord change, CLA assistant te demande de le signer à nouveau à ta pull request suivante.

## Règles de code

Les règles complètes sont dans [`docs/standards.md`](docs/standards.md) et les ADR de [`docs/adr/`](docs/adr). L'essentiel :

- **Tests d'abord.** Écris le test, vérifie qu'il échoue pour la bonne raison, écris le minimum de code qui le fait passer, puis remanie. Tests en tableau, fixtures réelles réduites dans `testdata/`, serveurs `httptest` pour les sources : un test qui a besoin du réseau est un bogue. Les paquets `domain` et `app` restent couverts à 90 % au moins.
- **Petites unités.** Une fonction fait une chose (moins de 40 lignes environ), un fichier porte un concept (moins de 300 lignes environ).
- **Pas de copie.** Avant d'écrire quelque chose, cherche dans `internal/` et étends ce qui existe. Pas de paquet fourre-tout `utils` ou `common`.
- **Erreurs** enveloppées avec leur contexte (`fmt.Errorf("sirene lookup %s: %w", siren, err)`), pas de `panic` hors de `main`.
- **Entrées-sorties** : `context.Context` en premier paramètre ; chaque appel sortant passe par le client HTTP de `internal/platform`, avec délai, taille maximale et liste d'hôtes autorisés.
- **Dépendances** : un module maintenu (actif dans les 6 derniers mois), sous licence compatible avec l'AGPL-3.0, sans vulnérabilité connue, nommé et justifié dans la pull request.
- Chaque fichier Go commence par `// SPDX-License-Identifier: AGPL-3.0-only`.

## Règles de rédaction des signaux

- Jamais « sûr » ni « fiable » : sans signal, la réponse est « Aucun signal connu dans nos sources ».
- Jamais « arnaque » à propos d'une entité nommée : un message énonce un fait daté et sourcé, par exemple « Figure sur la liste noire de l'AMF depuis le 08/10/2026 ».
- Une source n'est utilisée que si sa licence autorise la réutilisation commerciale ; elle est alors décrite dans [`docs/sources.md`](docs/sources.md).

## Vérifications locales

Avant chaque push, lance les mêmes commandes que la CI :

```sh
go build ./...
go vet ./...
golangci-lint run
go test -race -shuffle=on ./...
go tool govulncheck ./... tool
```

golangci-lint s'installe en binaire, à la version indiquée dans `docs/standards.md`. Sa configuration est dans `.golangci.yml` ; après l'avoir modifiée, valide-la :

```sh
golangci-lint config verify
golangci-lint fmt --diff
```

Si tu touches au chemin de `/check`, compare les benchmarks avec `main` :

```sh
go test -run '^$' -bench . -benchmem -count 6 ./internal/... > new.txt
go tool benchstat old.txt new.txt
```

Une régression de plus de 10 % se justifie dans la pull request.

## Règle des couches

Les dépendances vont vers l'intérieur uniquement ([ADR 0002](docs/adr/0002-architecture-propre.md)) : `domain` n'importe ni `app`, ni `adapter`, ni `platform` ; `app` n'importe ni `adapter` ni `platform` ; `platform` n'importe aucune autre couche ; personne n'importe `cmd`. `depguard` la vérifie. Pour t'assurer qu'elle est active, ajoute une violation volontaire, sans la commiter :

```sh
printf '// SPDX-License-Identifier: AGPL-3.0-only\n\npackage domain\n\nimport _ "github.com/arnakoff/api/internal/app"\n' > internal/domain/zz_violation.go
golangci-lint run --enable-only depguard   # doit échouer : import '…/internal/app' is not allowed from list 'domain'
rm internal/domain/zz_violation.go
```

## Commits et pull requests

- Branche : `<type>/<issue>-<description>`, par exemple `feat/12-domain-age`.
- Commits et titres de pull request au format [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/), à l'impératif, en minuscules, sans point final : `feat(signal): flag domains registered less than 30 days ago`. Le corps explique pourquoi. Portées : `domain`, `app`, `source`, `registry`, `httpapi`, `mcp`, `platform`, `sync`, `ci`, `deps`, `docs`. Un changement cassant prend `!` et un pied de page `BREAKING CHANGE:`.
- La description suit le [modèle de pull request](.github/pull_request_template.md) : `## Changes` (un point par comportement, textes visibles cités), `## Verification` (commandes lancées, benchmarks, mesures de qualité des signaux), puis `Part of #<épic>` et `Closes #<issue>`.
- Les pull requests sont fusionnées en squash dans `main` : leur titre devient le commit.
- Avant la fusion, la CI est verte et chaque conversation de relecture est résolue. Les personnes listées dans [`.github/CODEOWNERS`](.github/CODEOWNERS) sont ajoutées d'office en relecture.

## Labels

| Label | Usage |
|---|---|
| `type:feature` | Nouvelle fonctionnalité. |
| `type:bug` | Comportement incorrect. |
| `type:chore` | Outillage, CI, maintenance. |
| `type:docs` | Documentation. |
| `type:decision` | Décision à prendre, souvent une ADR. |
| `area:domain`, `area:app`, `area:source`, `area:registry`, `area:httpapi`, `area:mcp`, `area:platform`, `area:sync` | Couche ou paquet du code concerné. |
| `area:ci`, `area:deploy`, `area:docs`, `area:community` | Zone hors du code Go : CI, déploiement, documentation, fichiers communautaires. |
| `source` | Source de données, à ajouter ou à étudier. |
| `security` | Sécurité. |
| `owner-only` | Réservée au propriétaire : compte, paiement, juridique, DNS. |
| `good first issue` | Bonne première contribution. |
| `help wanted` | Aide bienvenue. |

Une épic s'intitule `[zone] ÉPIC — …` et regroupe ses sous-issues GitHub ; une sous-issue commence par `Partie de #<épic>.` et ses dépendances s'écrivent `Bloquée par #<issue>` dans sa section « Liens ».

## Versions et publication

Les versions suivent SemVer et sont produites par release-please à partir des titres de PR (Conventional Commits) fusionnés sur `main` : `fix` donne un correctif, `feat` une version mineure, `!` ou `BREAKING CHANGE:` une version majeure. Avant 1.0.0, un changement cassant ne monte que la version mineure (`bump-minor-pre-major`). La première version proposée est `v0.1.0` (`initial-version`).

Le workflow `.github/workflows/release-please.yml` tient à jour une PR de publication ; la fusionner crée le tag `vX.Y.Z`, la release GitHub et met à jour `CHANGELOG.md` et `.release-please-manifest.json`. Ces fichiers ne se modifient jamais à la main.

### Jeton (propriétaire)

Une PR ouverte avec `GITHUB_TOKEN` ne déclenche aucun workflow : la CI requise ne tournerait pas sur la PR de publication. Le workflow utilise donc le jeton d'une GitHub App de l'organisation, à créer une fois :

1. Créer une GitHub App dans l'organisation `arnakoff` (sans webhook), avec les permissions de dépôt *Contents*, *Pull requests* et *Issues* en écriture, puis l'installer sur `arnakoff/api` uniquement.
2. Enregistrer son *Client ID* dans la variable Actions `RELEASE_APP_CLIENT_ID` et une clé privée dans le secret Actions `RELEASE_APP_PRIVATE_KEY`.

Le jeton est créé à chaque exécution par `actions/create-github-app-token`, limité à ces trois permissions, et révoqué à la fin du job.

## Licence

En contribuant, tu acceptes que ta contribution soit publiée sous la [licence GNU AGPL-3.0-only](LICENSE) du dépôt, selon les termes de l'accord de contribution.
