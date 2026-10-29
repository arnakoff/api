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
