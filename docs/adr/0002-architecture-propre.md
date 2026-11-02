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
| Adaptateurs | `internal/adapter` | Sources, registres, listes de référence, stockage, HTTP entrant, MCP | `app`, `domain`, `platform` |
| Plateforme | `internal/platform` | Client HTTP sortant sécurisé, journalisation masquée, configuration | Bibliothèque standard, modules externes |
| Assemblage | `cmd/*` | Configuration et câblage, aucune logique | Tout |

- Les adaptateurs se rangent par famille :
  - `adapter/source/<nom>` télécharge une source amont, la lit et en tire un instantané validé avant de le publier (`cmd/sync`) ;
  - `adapter/registry/<nom>` interroge un registre à la demande, par identifiant, à travers la liste d'hôtes autorisés ;
  - `adapter/reference/<nom>` lit une liste de référence tenue dans le dépôt et embarquée dans le binaire (`go:embed`), chargée au démarrage. Rien n'est téléchargé : pas de synchronisation, pas d'instantané, donc pas de validation de taille ni de dernier instantané valide à conserver. Le contrôle passe au chargement : une ligne malformée fait échouer le chargement et la commande qui la charge s'arrête en erreur. La liste change à la main, par PR, chaque ligne vérifiée et la date du relevé dans la PR ; elle est décrite dans `docs/sources.md` comme une source ;
  - `adapter/store`, `adapter/httpapi` et `adapter/mcp` gardent leur rôle : stockage, HTTP entrant, MCP.
- La règle est vérifiée par `depguard` dans `golangci-lint` : une PR qui la viole échoue en CI.
- Les interfaces sont déclarées côté consommateur (`app`), petites, et implémentées dans `adapter`.
- Développement piloté par les tests (skill `tdd`), en partant de la couche la plus interne.

## Conséquences

- Le domaine et les cas d'usage se testent sans réseau ni horloge réelle, avec une couverture visée de 90 % et plus.
- Ajouter une source ne touche que `internal/adapter/source/<nom>` et le câblage dans `cmd/sync`.
- Ajouter une liste de référence ne touche que `internal/adapter/reference/<nom>`, sa section dans `docs/sources.md` et le câblage dans les commandes qui la chargent.
- Un peu plus de fichiers et d'interfaces qu'une approche directe : c'est le prix assumé de la maintenabilité.
