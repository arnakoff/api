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

## Précisions de #71 (03/11/2026)

Décisions du propriétaire à la mise en place du déploiement (`deploy/tofu/`, `.github/workflows/deploy.yml`, `docs/deploy.md`) :

- **Infrastructure en OpenTofu**, appliquée par le propriétaire depuis sa machine, jamais par la CI. État dans un bucket privé et versionné du projet (`arnakoff-tofu-state`), chiffré avant l'écriture par le chiffrement d'état d'OpenTofu, avec une phrase de passe que le propriétaire garde ; verrouillage par écriture conditionnelle S3 (`use_lockfile`), que Scaleway Object Storage prend en charge.
- **Secrets** : inchangé, ils restent dans l'environnement GitHub `production` et la CI les recopie à chaque déploiement, dans le conteneur et dans Secret Manager pour le job. OpenTofu ignore l'image et les secrets du conteneur une fois créé. Les mots de passe de la base sont en écriture seule (`password_wo`) : ils ne sont pas dans l'état. Les clés du bucket le sont, l'état est chiffré.
- **Migrations** : remplacé par #259, ci-dessous.
- **Pas de préproduction** pour l'instant.
- **Adresse du client** : derrière Serverless Containers, lue dans `X-Real-IP`, que Scaleway réécrit à chaque requête (#247) ; le point d'accès privé du conteneur reste fermé. **Autorité de la base** passée au conteneur en PEM (#249), faute de fichier monté.
- La CI obtient le droit `SecretManagerSecretWrite` sur le projet, pour écrire les versions des secrets du job.

## Migrations depuis le réseau privé (#259, 04/11/2026)

Au premier déploiement, le point d'accès public de la base (`51.159.27.181:3162`) n'avait pas de nom d'hôte et le certificat de l'instance ne le couvrait pas (SAN : l'adresse privée `172.16.16.2` et `212.47.251.84`) : `verify-full` échouait. Décision du propriétaire :

- Les migrations tournent dans le réseau privé, dans un conteneur Serverless `migrate` (Serverless Jobs ne rejoint pas encore de réseau privé, FAQ de Scaleway), créé par OpenTofu avec `migrate=true` et retiré à l'`apply` suivant. Il lance `admin migrate -listen`, qui répond `200` sur `/readyz` une fois les migrations appliquées. La base n'a plus jamais de point d'accès public.
- Les identifiants du rôle administrateur viennent d'OpenTofu au moment de l'`apply` : le mot de passe est dans l'état, chiffré sur la machine du propriétaire, tant que le conteneur existe, et n'est jamais dans la CI.
- Les migrations restent un geste du propriétaire, avant le déploiement de la version qui les apporte.

## Sauvegarde des clés hors de Scaleway (#75, 05/11/2026)

Les sauvegardes automatiques de la base sont des instantanés, que l'on ne télécharge pas, et créer ou exporter une sauvegarde logique demande `RelationalDatabasesFullAccess`, qui permet aussi de supprimer la base. Décision du propriétaire, avec le moins de droits possible :

- Un conteneur Serverless privé `backup`, sur le réseau privé, lancé chaque jour par un déclencheur cron, lit `api_keys` avec un rôle `readonly` (`admin backup -listen`). Aucune clé d'API Scaleway n'intervient.
- Le dump est chiffré avec age pour la clé publique du propriétaire, puis ajouté dans un bucket Amazon S3 à Paris (`eu-west-3`, `deploy/tofu-aws`), par une clé qui ne peut qu'ajouter un objet nouveau. La copie est gardée 30 jours (ADR 0006).
- AWS est un second hébergeur, américain : il ne reçoit que du chiffré, et ce transfert va au registre des traitements (#74).

## Conséquences

- Coût estimé : Managed Database ≈ 11,40 € (0,0136 + 0,002 €/h) ; conteneur `api` toujours actif, 0,25 vCPU et 512 Mo, ≈ 6,40 € après l'offre gratuite mensuelle (200 000 vCPU-s et 400 000 Go-s), ≈ 13 € avec 0,5 vCPU ; Job, Object Storage et Container Registry de l'ordre de quelques euros, à vérifier.
- Une issue ajoute l'adaptateur Object Storage du port `SnapshotStore`, avec son module S3 choisi et vérifié dans sa PR (licence, maintenance, `govulncheck`).
- `compose.yaml` reste l'environnement local ; il n'est pas le modèle de la production.
- #71 déploie avec la CLI ou l'API Scaleway ; #72 et #75 s'appuient sur cette ADR.
- Aucun serveur à administrer ; chaque instance d'`api` garde sa propre limite par minute (ADR 0004).
