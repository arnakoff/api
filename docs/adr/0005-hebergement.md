# 0005 — Hébergement de production

- **Statut** : accepté
- **Date** : 03/11/2026

## Contexte

L'API doit tourner dans l'Union européenne (ADR 0001), avec PostgreSQL géré pour les clés et les quotas (ADR 0004). L'image publiée par `.github/workflows/image.yml` est signée et se déploie par digest après vérification de sa signature (#69, #71). Deux processus tournent en continu sur la même image : `api` (HTTP) et `sync`, qui télécharge les sources et écrit les instantanés que `api` relit chaque minute. Aujourd'hui, ils partagent un volume (`compose.yaml`).

Critères (#70) : conteneurs gérés, stockage persistant, IPv6, TLS, prix, localisation des données, siège dans l'Union européenne.

## Options comparées

Prix hors taxes relevés le 03/11/2026 sur les pages officielles, les API de catalogue (Scaleway, OVHcloud, Clever Cloud) et la documentation des hébergeurs.

| Option | Contraintes du projet | PostgreSQL géré | Coût mensuel estimé | Siège |
|---|---|---|---|---|
| Scaleway, Serverless Containers et Jobs + Object Storage | Oui, si les instantanés passent du disque à Object Storage | Oui (fr-par) | ≈ 18 à 25 € | Paris (groupe Iliad) |
| Scaleway, machine DEV1-S avec Docker Compose | Oui, sans changement de code ; serveur à administrer | Oui (fr-par) | ≈ 18 à 22 € | Paris (groupe Iliad) |
| Clever Cloud | FS Bucket indisponible pour les applications Docker | Oui | ≈ 16 € | Nantes |
| OVHcloud, VPS-1 avec Docker Compose | Oui ; serveur à administrer | Oui, dès ≈ 54 € | ≈ 59 € | Roubaix |
| Hetzner | Pas de PostgreSQL géré | Non | — | Allemagne |
| Exoscale | Oui ; aucune zone en France | Oui | ≈ 60 € | Lausanne (hors UE) |
| Scalingo, Koyeb, Upsun, Alwaysdata | Pas de volume partagé entre deux processus, ou pas de déploiement d'image | — | — | France |

## Décision

Le propriétaire choisit **Scaleway, région fr-par, en mode serverless** : aucun serveur à administrer.

- **`api`** : Serverless Container, `min-scale` à 1 au moins pour garder les instantanés et la limite par minute en mémoire, sonde HTTP sur `/readyz`, domaine `api.arnakoff.com` avec certificat Let's Encrypt géré par Scaleway.
- **`sync`** : Serverless Job planifié par cron (fuseau `Europe/Paris`), qui lance `sync -once` sur la même image.
- **Instantanés** : un bucket Object Storage fr-par, versionné, remplace le volume partagé. Un nouvel adaptateur du port `SnapshotStore` (`Save`, `Load`) dans `internal/adapter/store` lit et écrit les objets par l'API S3 (`s3.fr-par.scw.cloud`, dans la liste des hôtes autorisés du client HTTP sortant). La validation et le refus d'un instantané qui rétrécit de plus de 20 % restent dans `app`, inchangés. `api` relit les objets avec une requête conditionnelle (ETag) pour ne télécharger que ce qui a changé. Cette décision remplace, pour la production, le stockage des instantanés sur disque de l'ADR 0001 ; `store.Disk` reste pour le développement et les tests.
- **Clés d'accès** : une clé IAM pour `sync`, en lecture et écriture sur ce seul bucket, et une pour `api`, en lecture seule, restreintes par une politique de bucket.
- **PostgreSQL** : Managed Database fr-par, plus petit nœud (DB-DEV-S : 2 vCPU, 2 Go de RAM), sauvegarde automatique quotidienne conservée 7 jours, plus la copie hors de Scaleway prévue par #75. `api` y accède par un Private Network, en TLS.
- **Image** : la CI vérifie la signature cosign de l'image publiée sur GHCR, copie son manifeste `linux/amd64` dans Scaleway Container Registry, puis déploie ce digest. Scaleway déconseille les registres externes à cause de leurs limites de débit, et ses conteneurs n'acceptent que `linux/amd64`.
- **Secrets** : Scaleway ne propose ni fédération OIDC pour GitHub Actions ni identité attachée à un conteneur ou à un job ; une clé d'API existe donc toujours quelque part. Toute clé expire (plafond de durée fixé pour l'organisation). La clé de `api` est une variable d'environnement secrète du conteneur, seule voie possible : les Serverless Containers ne sont pas reliés à Secret Manager. La clé de `sync` est une référence Secret Manager montée en fichier dans le job. Les secrets restent dans les environnements GitHub et la CI les recopie chez Scaleway au déploiement (#71).
- **Clé de la CI** : application IAM `ci-deploy`, droits `ContainersFullAccess`, `ServerlessJobsFullAccess` et `ContainerRegistryFullAccess` limités au projet `arnakoff-production` ; clé de 30 jours dans les secrets `SCW_ACCESS_KEY` et `SCW_SECRET_KEY` de l'environnement GitHub `production`, qui exige la revue du propriétaire et n'accepte que `main`. Pas de fédération OIDC : la rotation est manuelle (#238). Les droits Secret Manager s'ajoutent quand #71 en a besoin.

## Constats à l'ouverture du compte (03/11/2026)

Relevés avec la CLI Scaleway 2.65.2 sur le compte ouvert, et dans les sources de la documentation Scaleway (`github.com/scaleway/docs-content`).

- **PostgreSQL** : les moteurs proposés en fr-par s'arrêtent à PostgreSQL 17 (`scw rdb engine list`). La CI passe de la 18 à la 17. DB-DEV-S est toujours en vente, en stock, mais de première génération (`scw rdb node-type list`) ; son prix n'apparaît pas dans la liste, à vérifier.
- **Migrations (`admin`)** : la documentation le confirme, les Serverless Jobs ne se rattachent pas à un Private Network. Le choix reste ouvert : soit un point d'accès public de la base, limité par ACL, en TLS et avec le seul rôle de migration, soit une autre façon de lancer `admin` depuis le réseau privé. À trancher dans #71.
- **Conteneurs** : la documentation associe 250 mvCPU à 512 Mo.
- **Restent à vérifier** : le déploiement par digest (`@sha256:`) depuis Container Registry (non documenté ; à tester au premier déploiement), le partage ou non de l'offre gratuite entre conteneurs et jobs, et la période de facturation d'Object Storage et de Container Registry.

## Conséquences

- Coût estimé : Managed Database ≈ 11,40 € (0,0136 + 0,002 €/h) ; conteneur `api` toujours actif, 0,25 vCPU et 512 Mo, ≈ 6,40 € après l'offre gratuite mensuelle (200 000 vCPU-s et 400 000 Go-s), ≈ 13 € avec 0,5 vCPU ; Job, Object Storage et Container Registry de l'ordre de quelques euros, à vérifier.
- Une issue ajoute l'adaptateur Object Storage du port `SnapshotStore`, avec son module S3 choisi et vérifié dans sa PR (licence, maintenance, `govulncheck`).
- `compose.yaml` reste l'environnement local ; il n'est pas le modèle de la production.
- #71 déploie avec la CLI ou l'API Scaleway ; #72 et #75 s'appuient sur cette ADR.
- Aucun serveur à administrer ; chaque instance d'`api` garde sa propre limite par minute (ADR 0004).
