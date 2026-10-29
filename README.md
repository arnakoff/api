# Arnakoff API

[![CI](https://github.com/arnakoff/api/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/arnakoff/api/actions/workflows/ci.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/arnakoff/api/badge)](https://scorecard.dev/viewer/?uri=github.com/arnakoff/api)
[![Licence : AGPL-3.0-only](https://img.shields.io/badge/licence-AGPL--3.0--only-blue)](LICENSE)

API publique anti-arnaque : pour un site, un e-mail, un numéro de téléphone, un IBAN ou un SIREN, elle renvoie des signaux sourcés expliqués en français.

## Statut

En construction : aucune version n'est publiée et l'API n'est pas encore en service. Les commandes de `cmd/` ne font encore rien. Suis l'avancement dans la [feuille de route](https://github.com/orgs/arnakoff/projects/1).

## Ce que fait l'API

Tu lui donnes une entrée ; elle la confronte à des sources publiques (listes noires de l'AMF et de l'ACPR, répertoire SIRENE, BODACC, âge du domaine, numérotation de l'ARCEP…) et renvoie une liste de signaux. Chaque signal porte :

- `code` : identifiant stable du signal ;
- `level` : niveau du signal ;
- `message` : une phrase factuelle en français, par exemple « Figure sur la liste noire de l'AMF depuis le 08/10/2026 » ;
- `source` et `source_url` : l'éditeur de la donnée et où la vérifier ;
- `observed_at` : la date à laquelle la donnée a été relevée.

Les signaux prévus sont dans [`docs/signals.md`](docs/signals.md), les sources et leurs licences dans [`docs/sources.md`](docs/sources.md). L'API sera aussi exposée comme serveur [MCP](https://modelcontextprotocol.io).

## Principes

- **Jamais « sûr ».** Sans signal, la réponse est « Aucun signal connu dans nos sources » : l'absence d'une liste noire ne prouve rien.
- **Des faits, pas des accusations.** Chaque signal cite sa source ; aucun message ne qualifie une entité nommée d'arnaque.
- **Des licences d'abord.** Une source n'est utilisée que si sa licence autorise la réutilisation commerciale.
- **Les données personnelles protégées.** Les e-mails, numéros et IBAN ne sont jamais journalisés en clair ni transmis à un tiers.
- **Aucune URL fournie par l'utilisateur n'est appelée.** Les registres sont interrogés par identifiant, à travers une liste d'hôtes autorisés.
- **Une synchronisation prudente.** Un instantané de source invalide, ou qui perd plus de 20 % de ses entrées, est rejeté ; le précédent reste en service.

## Démarrer

Il te faut Go 1.27 : la version exacte (`toolchain go1.27.3`) est dans `go.mod` et se télécharge d'elle-même. Installe aussi [golangci-lint](https://golangci-lint.run) à la version indiquée dans [`docs/standards.md`](docs/standards.md).

```sh
go build ./...
go vet ./...
golangci-lint run
go test -race -shuffle=on ./...
go tool govulncheck ./... tool
```

La CI lance les mêmes commandes sur chaque pull request. Les tests n'utilisent jamais le réseau.

## Organisation

Architecture propre ([ADR 0002](docs/adr/0002-architecture-propre.md)) : les dépendances vont vers l'intérieur uniquement.

| Dossier | Rôle |
|---|---|
| `internal/domain/` | Objets métier, normalisation des entrées et règles des signaux, sans entrée-sortie. |
| `internal/app/` | Cas d'usage (vérification, synchronisation) et interfaces dont ils ont besoin. |
| `internal/adapter/` | Sources, registres, stockage, API HTTP et serveur MCP. |
| `internal/platform/` | Client HTTP sortant, journaux, configuration. |
| `cmd/` | Points d'entrée : `api`, `sync`, `mcp` et `eval`. |

## Documentation

- Décisions d'architecture : [0001 — Socle technique](docs/adr/0001-socle-technique.md), [0002 — Architecture propre](docs/adr/0002-architecture-propre.md), [0003 — Licences](docs/adr/0003-licences.md).
- [Standards et versions](docs/standards.md), [sources](docs/sources.md), [signaux](docs/signals.md).
- Le site et la documentation publique de l'API sont dans [`arnakoff/web`](https://github.com/arnakoff/web).

## Contribuer

Lis [CONTRIBUTING.md](CONTRIBUTING.md). Pour une faille de sécurité, suis la [politique de sécurité](https://github.com/arnakoff/.github/blob/main/SECURITY.md), jamais une issue publique. Pour de l'aide, vois [SUPPORT.md](https://github.com/arnakoff/.github/blob/main/SUPPORT.md).

## Licence

Le code est sous [GNU AGPL-3.0-only](LICENSE) ([ADR 0003](docs/adr/0003-licences.md)). Les données gardent la licence de leur source, indiquée dans [`docs/sources.md`](docs/sources.md).
