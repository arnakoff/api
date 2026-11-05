# 0006 — Durées de conservation des données personnelles

- **Statut** : accepté
- **Date** : 03/11/2026

## Contexte

Les e-mails, numéros de téléphone et IBAN sont des données personnelles (RGPD) ; le libellé d'une clé API et l'adresse du compte qui la demande (#59) aussi. Le RGPD demande une durée de conservation fixée pour chaque donnée personnelle, puis sa suppression. L'ADR 0004 renvoie à cette décision les durées des clés révoquées, et #59 demande que l'adresse e-mail d'un compte soit gardée selon elle.

Ce que l'API garde aujourd'hui :

- la table `api_keys` (ADR 0004) : l'empreinte HMAC de chaque clé, jamais la clé en clair, son environnement, son libellé, ses quatre derniers caractères, ses dates de création et de révocation ;
- les journaux applicatifs : méthode, route, statut, taille, durée et identifiant de requête. Ni adresse IP ni entrée vérifiée en clair : le journal remplace un e-mail, un numéro ou un IBAN par une forme masquée et un pseudonyme HMAC à clé (`internal/platform/logging`) ;
- en mémoire seulement : les réponses des registres (RDAP, SIRENE, BODACC), gardées 24 heures, et les compteurs de limite de débit, sur une minute.

## Décision

Le propriétaire fixe les durées suivantes.

| Donnée | Durée | Mise en œuvre |
|---|---|---|
| Entrées vérifiées (domaine, e-mail, numéro, IBAN, SIREN) | Aucune conservation | Jamais écrites en base ni journalisées en clair (ADR 0004, `CLAUDE.md`) |
| Clé active : empreinte, libellé, dates | Tant que la clé est active | — |
| Clé révoquée, et adresse e-mail du compte qui la porte (#59) | 1 an (365 jours) après la révocation | Purge automatique, ci-dessous |
| Demande de lien de connexion (#59) : adresse e-mail et jeton | 24 heures après l'expiration ou l'utilisation du lien | Purge ajoutée par #59 |
| Journaux applicatifs | 6 mois | Rétention réglée dans l'outil de journaux de l'hébergeur (#72) |
| Réponses des registres, compteurs de limite de débit | 24 heures, une minute | En mémoire, perdus au redémarrage |
| Copie des clés hors de Scaleway (#75) : table `api_keys`, adresses e-mail comprises, chiffrée avec age | 30 jours, supprimée au plus tard le 31ᵉ | Règle de cycle de vie du bucket Amazon S3 (`deploy/tofu-aws`) ; les demandes de lien n'y sont pas copiées |

- **Purge des clés révoquées** : le processus `api` supprime au démarrage, puis une fois par jour, les lignes de `api_keys` révoquées depuis plus de 365 jours (`app.KeyPurge`, `domain.RevokedKeyRetention`). Plusieurs instances peuvent la lancer en même temps : la requête `DELETE` est idempotente. Un échec est journalisé et la purge recommence le lendemain. Le rôle PostgreSQL de l'API doit donc avoir le droit `DELETE` sur `api_keys`, en plus de la lecture et de l'écriture prévues par l'ADR 0004.
- **Une seule source de vérité** : la durée vit dans le domaine (`domain.RevokedKeyRetention`) ; la documentation et le registre des traitements (#74) la reprennent sans la redéfinir.

## Conséquences

- #59 garde l'adresse du compte avec sa clé, la supprime avec elle, et ajoute la purge des demandes de lien après 24 heures.
- #72 règle à 6 mois la rétention des journaux chez Scaleway.
- Une ligne supprimée reste dans les sauvegardes jusqu'à leur propre expiration : 7 jours pour la sauvegarde automatique de la base (ADR 0005). La copie hors de Scaleway la garde 30 jours de plus (#75), soit au plus 365 + 31 jours pour une clé révoquée.
- #74 reprend ces durées dans le registre des traitements.
- Une nouvelle donnée personnelle conservée demande de modifier cette ADR avant d'être stockée.
