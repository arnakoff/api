# Contribuer à Arnakoff

## Vérifications locales

Avant chaque push, lancez les mêmes commandes que la CI :

```sh
go build ./...
go vet ./...
golangci-lint run
go test -race -shuffle=on ./...
govulncheck ./...
```

golangci-lint s'installe en binaire, à la version indiquée dans `docs/standards.md`. Sa configuration est dans `.golangci.yml` ; après l'avoir modifiée, validez-la :

```sh
golangci-lint config verify
golangci-lint fmt --diff
```

## Règle des couches

Les dépendances vont vers l'intérieur uniquement (ADR 0002) : `domain` n'importe ni `app`, ni `adapter`, ni `platform` ; `app` n'importe ni `adapter` ni `platform` ; `platform` n'importe aucune autre couche ; personne n'importe `cmd`. `depguard` la vérifie. Pour s'assurer qu'elle est active, ajoutez une violation volontaire, sans la commiter :

```sh
printf '// SPDX-License-Identifier: AGPL-3.0-only\n\npackage domain\n\nimport _ "github.com/arnakoff/api/internal/app"\n' > internal/domain/zz_violation.go
golangci-lint run --enable-only depguard   # doit échouer : import '…/internal/app' is not allowed from list 'domain'
rm internal/domain/zz_violation.go
```

## Versions et publication

Les versions suivent SemVer et sont produites par release-please à partir des titres de PR (Conventional Commits) fusionnés sur `main` : `fix` donne un correctif, `feat` une version mineure, `!` ou `BREAKING CHANGE:` une version majeure. Avant 1.0.0, un changement cassant ne monte que la version mineure (`bump-minor-pre-major`). La première version proposée est `v0.1.0` (`initial-version`).

Le workflow `.github/workflows/release-please.yml` tient à jour une PR de publication ; la fusionner crée le tag `vX.Y.Z`, la release GitHub et met à jour `CHANGELOG.md` et `.release-please-manifest.json`. Ces fichiers ne se modifient jamais à la main.

### Jeton (propriétaire)

Une PR ouverte avec `GITHUB_TOKEN` ne déclenche aucun workflow : la CI requise ne tournerait pas sur la PR de publication. Le workflow utilise donc le jeton d'une GitHub App de l'organisation, à créer une fois :

1. Créer une GitHub App dans l'organisation `arnakoff` (sans webhook), avec les permissions de dépôt *Contents*, *Pull requests* et *Issues* en écriture, puis l'installer sur `arnakoff/api` uniquement.
2. Enregistrer son *Client ID* dans la variable Actions `RELEASE_APP_CLIENT_ID` et une clé privée dans le secret Actions `RELEASE_APP_PRIVATE_KEY`.

Le jeton est créé à chaque exécution par `actions/create-github-app-token`, limité à ces trois permissions, et révoqué à la fin du job.
