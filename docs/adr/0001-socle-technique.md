# 0001 — Socle technique

- **Statut** : accepté
- **Date** : 29/10/2026

## Contexte

Arnakoff agrège des sources publiques françaises (listes noires AMF et ACPR, SIRENE, BODACC, RDAP, ARCEP) pour rendre des signaux sourcés sur un site, un e-mail, un numéro, un IBAN ou un SIREN. L'API est publique, gratuite avec quota, puis payante pour les intégrateurs. Les requêtes contiennent des données personnelles.

## Décision

- **Langage** : Go 1.27 (`go 1.27.0`, `toolchain go1.27.3` dans `go.mod`, version vérifiée le 29/10/2026 ; voir `docs/standards.md`). Bibliothèques externes maintenues plutôt que des réimplémentations (IBAN, numéros de téléphone, JSON Schema, SDK MCP).
- **Dépôts** : organisation GitHub `arnakoff`. `arnakoff/api` (ce dépôt : API, synchronisation, serveur MCP), `arnakoff/web` (site et documentation). Plus tard, si besoin : `arnakoff/extension`, `arnakoff/bot`, `arnakoff/.github` (profil de l'organisation).
- **Données** : les listes (AMF, ACPR, ARCEP) sont téléchargées par `cmd/sync`, validées, puis servies depuis un instantané local en mémoire. Les registres consultés à la demande (SIRENE, BODACC, RDAP) passent par un cache. La requête de l'utilisateur ne quitte l'infrastructure que sous la forme d'un identifiant public (domaine, SIREN), jamais un e-mail, un numéro ou un IBAN.
- **Contrat** : `api/openapi.yaml` fait foi ; chemin versionné `/v1`.
- **Hébergement** : conteneur Docker, hébergeur situé dans l'Union européenne (RGPD). Choix précis à trancher dans une ADR dédiée.

## Conséquences

- Pas de base de données au démarrage : les clés API et les quotas pourront demander PostgreSQL ensuite (ADR à venir).
- La qualité dépend de la fraîcheur des instantanés : `check-sources` et les alertes de synchronisation sont nécessaires dès la première version.
- La licence de la liste ACPR doit être obtenue avant l'ouverture commerciale.
