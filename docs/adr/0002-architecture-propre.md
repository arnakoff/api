# 0002 — Architecture propre

- **Statut** : accepté
- **Date** : 29/10/2026

## Contexte

Arnakoff vise une qualité premium : code propre, sécurisé, testé et mesuré. Les sources de données changeront souvent (formats, nouveaux registres), alors que les règles métier (signaux, normalisation) doivent rester stables et testables sans réseau.

## Décision

Architecture propre en quatre couches, dépendances orientées vers l'intérieur uniquement :

| Couche | Dossier | Contient | Peut importer |
|---|---|---|---|
| Domaine | `internal/domain` | Objets valeur (`Domain`, `Email`, `Phone`, `IBAN`, `SIREN`), `Signal`, `Level`, `Snapshot`, règles de signaux, normalisation | Bibliothèque standard, bibliothèques de parsing validées |
| Application | `internal/app` | Cas d'usage (`Check`, `Sync`) et ports (interfaces) qu'ils consomment | `domain` |
| Adaptateurs | `internal/adapter` | Sources, registres, stockage, HTTP entrant, MCP | `app`, `domain`, `platform` |
| Plateforme | `internal/platform` | Client HTTP sortant sécurisé, journalisation masquée, configuration | Bibliothèque standard, modules externes |
| Assemblage | `cmd/*` | Configuration et câblage, aucune logique | Tout |

- La règle est vérifiée par `depguard` dans `golangci-lint` : une PR qui la viole échoue en CI.
- Les interfaces sont déclarées côté consommateur (`app`), petites, et implémentées dans `adapter`.
- Développement piloté par les tests (skill `tdd`), en partant de la couche la plus interne.

## Conséquences

- Le domaine et les cas d'usage se testent sans réseau ni horloge réelle, avec une couverture visée de 90 % et plus.
- Ajouter une source ne touche que `internal/adapter/source/<nom>` et le câblage dans `cmd/sync`.
- Un peu plus de fichiers et d'interfaces qu'une approche directe : c'est le prix assumé de la maintenabilité.
