# 0007 — Rotation automatique des clés Scaleway

- **Statut** : accepté
- **Date** : 05/11/2026

## Contexte

Les clés API Scaleway durent au plus 90 jours, plafond fixé pour l'organisation (relevé le 04/11/2026). Trois clés de production sont aujourd'hui renouvelées à la main (ADR 0005) :

- celles des applications `arnakoff-api` et `arnakoff-sync`, qui lisent et écrivent le bucket des instantanés. OpenTofu les crée (`deploy/tofu/storage.tf`) et les recrée quand `key_expiry` change, ce qui coupe l'accès au bucket entre le `tofu apply` et le déploiement suivant (#254) ;
- celle de l'application `ci-deploy`, valable 30 jours, gardée dans les secrets `SCW_ACCESS_KEY` et `SCW_SECRET_KEY` de l'environnement GitHub `production` (#238).

Une échéance manquée arrête la lecture des instantanés ou tout déploiement. Scaleway ne propose ni fédération OIDC pour GitHub Actions, ni identité de charge de travail sans clé pour les Serverless Containers et Jobs (ADR 0005, revérifié avec `scw` 2.65.2 le 05/11/2026). Le propriétaire veut une rotation sans aucune action manuelle (décision du 05/11/2026).

## Ce qui a été vérifié (05/11/2026)

- **Droits IAM** (`scw iam permission-set list`, organisation `arnakoff`) :
  - le plus petit droit qui gère les clés d'une application est `IAMApplicationManager`, de portée `organization` : « Full access to IAM applications, including management of applications API keys ». Il ne se restreint pas à une liste d'applications ; `IAMManager`, plus large, a été écarté ;
  - Secret Manager : `SecretManagerSecretCreate` crée une version, `SecretManagerSecretWrite` désactive la précédente, `SecretManagerSecretAccess` lit une valeur en clair. Les trois sont de portée projet ;
  - Serverless Containers : pas de droit d'écriture plus étroit que `ContainersFullAccess` (projet). Seuls existent en plus `ContainersReadOnly` et `ContainersPrivateAccess`.
- **Secrets d'un conteneur** :
  - un Serverless Container ne référence pas Secret Manager. La documentation dit : « Secrets on Serverless Containers are not linked with Secret Manager product yet » (`scaleway/docs-content`, `pages/serverless-containers/concepts.mdx`). Ses secrets restent des `secret_environment_variables` ;
  - un changement de paramètre redéploie le conteneur sans coupure : le trafic passe à la nouvelle version une fois prête (`concepts.mdx`, `faq.mdx`).
- **Mise à jour partielle des secrets d'un conteneur** :
  - l'API `containers/v1beta1` fusionne par clé : « secret environment variables that are not specified in this field will be kept unchanged » (`github.com/scaleway/scaleway-sdk-go` v1.39.0, `api/container/v1beta1/container_sdk.go`, `UpdateContainerRequest`) ;
  - l'API `containers/v1`, qu'utilise `scw` 2.65.2, prend une table complète. Scaleway n'en documente pas le comportement clé par clé, et le fournisseur Terraform la traite comme un remplacement complet (`internal/services/container/helpers_container.go`). « À vérifier » avant de s'y fier.
- **SDK Go de Scaleway** :
  - `github.com/scaleway/scaleway-sdk-go` v1.39.0, du 12/10/2026, sous licence Apache-2.0, sans vulnérabilité connue (`vuln.go.dev`) ;
  - il accepte un client HTTP fourni (`scw.WithHTTPClient`), donc celui de `internal/platform`, et appelle `api.scaleway.com` ;
  - une clé se crée avec une date d'expiration (`CreateAPIKeyRequest.ExpiresAt`).
- **Audit Trail** :
  - il journalise `CreateAPIKey`, `UpdateAPIKey` et `DeleteAPIKey` (`pages/audit-trail/reference-content/adt-supported-endpoints.mdx`) ;
  - il envoie lui-même des alertes par e-mail, dont les préconfigurées `ApiKeyCreated`, `ApiKeyDeleted` et `ApiKeyCreationUnauthorized`. Cockpit, lui, ne lit pas l'Audit Trail (`cockpit-product-integration.mdx` : « Not integrated »).
- **GitHub** :
  - un secret d'environnement s'écrit par `PUT /repos/{owner}/{repo}/environments/{environment_name}/secrets/{secret_name}`, chiffré par une boîte scellée libsodium avec la clé publique de l'environnement (`GET …/secrets/public-key`) ;
  - une GitHub App y accède avec la seule permission de dépôt « Environments: write » (docs.github.com, « Permissions required for GitHub Apps ») ;
  - le jeton d'installation dure une heure et se restreint à un dépôt et à des permissions données (`POST /app/installations/{installation_id}/access_tokens`). Le JWT de l'App est signé en RS256 et vit au plus 10 minutes ;
  - une clé privée d'App n'expire pas. Une App peut en avoir jusqu'à 25, ce qui permet de la changer sans coupure, mais seulement depuis l'interface de GitHub ;
  - la boîte scellée se fait avec `golang.org/x/crypto/nacl/box.SealAnonymous` (v0.58.0, BSD-3-Clause, déjà présent en dépendance indirecte), et le JWT avec `crypto/rsa` de la bibliothèque standard ;
  - tous ces appels vont à `api.github.com`.

## Décision

### Un job `rotate`

Un Serverless Job privé `rotate`, sur l'image de l'API, sans point d'accès entrant. Il est lancé chaque jour à 03:10, heure de Paris, en dehors des passages de `sync` (`17 */6 * * *`, 30 minutes au plus). Les clés d'une application sont renouvelées dès que l'une d'elles a plus de 30 jours, ou expire dans moins de 59 jours, comme la clé de 30 jours de `ci-deploy` faite à la main. Une clé créée par un passage en échec est plus récente que la clé en service, qui reste donc due (précisé dans #307 et #308). Les nouvelles clés valent 89 jours : un échec est repris le lendemain, et il reste près de deux mois avant l'expiration.

Pour chaque application dont la clé doit changer, dans cet ordre :

1. **`arnakoff-api` et `arnakoff-sync`** :
   - le job crée la nouvelle clé et vérifie qu'elle lit le bucket des instantanés ;
   - il écrit celle de `sync` dans Secret Manager, en désactivant la version précédente ;
   - il met à jour les deux secrets du bucket du conteneur `api` par l'API `containers/v1beta1`, qui laisse les autres secrets intacts, puis attend que le conteneur soit prêt ;
   - il supprime l'ancienne clé de chaque application.
2. **`arnakoff-mail`** (ajoutée par #295, le 05/11/2026) :
   - le job crée la nouvelle clé et vérifie qu'elle s'authentifie auprès du serveur SMTP de Scaleway Transactional Email (`EHLO`, `AUTH PLAIN`, `QUIT`, sans envoyer de courrier) ;
   - il l'écrit dans le secret `ARNAKOFF_SMTP_PASSWORD` du conteneur `api` par l'API `containers/v1beta1`, puis attend que le conteneur soit prêt. L'identifiant SMTP, le projet, ne change pas ;
   - il supprime l'ancienne clé.
3. **`ci-deploy`** :
   - le job crée la nouvelle clé ;
   - il l'écrit dans `SCW_ACCESS_KEY` et `SCW_SECRET_KEY` de l'environnement GitHub `production`, par la GitHub App ;
   - il supprime l'ancienne clé. Un déploiement en cours à ce moment échoue et se relance.
4. **Sa propre clé** :
   - le job lit sa clé dans Secret Manager, en crée une nouvelle et l'écrit comme nouvelle version de son secret, en désactivant la précédente ;
   - il supprime l'ancienne clé en tout dernier.

Une ancienne clé n'est supprimée qu'après que la nouvelle est en place et vérifiée. Une étape qui échoue arrête le job sans rien supprimer : il ne reste alors qu'une clé en trop, reprise au passage suivant.

### Liste fermée

Les identifiants des cinq applications sont une liste fermée, fixée par la configuration du job et vérifiée par le code. Le job refuse toute autre application, et refuse de toucher une clé dont l'application n'est pas dans la liste.

### Exception au moindre privilège

Le job reçoit `IAMApplicationManager` sur l'organisation, car aucun droit plus étroit ne gère les clés d'une application. Il peut donc créer une clé pour n'importe quelle application de l'organisation, `ci-deploy` comprise. Une fuite de sa clé donnerait au moins les droits de déploiement de `ci-deploy`.

Il reçoit aussi `ContainersFullAccess` sur le projet, pour écrire les secrets du conteneur `api`, ainsi que `SecretManagerSecretCreate`, `SecretManagerSecretWrite` et `SecretManagerSecretAccess` sur le projet. Le propriétaire a accepté cet échange le 05/11/2026 contre la fin de toute rotation manuelle.

Garde-fous :

- l'image est construite et signée par la CI, puis déployée par empreinte (ADR 0005) ;
- le job n'a aucun point d'accès entrant et ne lit aucune entrée extérieure ;
- ses appels sortants passent par le client HTTP de `internal/platform`, limité à `api.scaleway.com`, `api.github.com` et au point d'accès S3 du bucket, avec délais et taille de réponse bornés ;
- ses journaux ne contiennent jamais un secret : ni clé secrète, ni clé privée d'App, ni jeton. Seuls l'identifiant d'accès public d'une clé et son application y figurent ;
- un échec est un passage en erreur, signalé par l'alerte Cockpit `JobRunFailed` déjà active (#72) ;
- les alertes préconfigurées de l'Audit Trail `ApiKeyCreated`, `ApiKeyDeleted` et `ApiKeyCreationUnauthorized` sont activées. Une création de clé en dehors du passage de 03:10 est à examiner.

### GitHub App

Une GitHub App appartenant à l'organisation `arnakoff`, installée sur `arnakoff/api` seul, avec la seule permission de dépôt « Environments: write ». Sa clé privée est un secret Secret Manager, lu par le job seul.

Le job demande un jeton d'installation restreint à `arnakoff/api` et à `environments: write`. Il signe le JWT avec la bibliothèque standard et chiffre les secrets avec `golang.org/x/crypto/nacl/box`.

La clé privée de l'App n'expire pas et ne se change que depuis l'interface de GitHub : c'est la seule clé longue qui reste, en échange de plus aucune rotation manuelle. Elle se révoque dans l'interface à la moindre fuite.

### Amorçage

Une seule action manuelle, faite par le propriétaire :

- créer la GitHub App et l'installer sur `arnakoff/api` ;
- écrire sa clé privée et la première clé du job dans Secret Manager ;
- appliquer OpenTofu.

Ensuite, le propriétaire n'agit que sur alerte.

## À vérifier avant la mise en production

À vérifier dans #307 et #308 :

- qu'une application puisse créer et supprimer sa propre clé avec `IAMApplicationManager` ;
- qu'une clé supprimée cesse aussitôt de fonctionner. Le job ne s'appuie pas dessus pour la continuité ;
- que `IAMApplicationManager` ne permette pas de modifier les politiques IAM, sinon le job pourrait s'élargir ses propres droits ;
- comment `containers/v1` traite les clés absentes de `secret_environment_variables`. Le déploiement (`.github/workflows/deploy.yml`, `scw` 2.65.2) passe par cette API : s'il remplace la table entière, un déploiement qui n'envoie plus les clés du bucket les effacerait ;
- que l'organisation, sur le plan gratuit de GitHub, voie les événements `environment.update_actions_secret` dans son journal d'audit.

## Conséquences

- #307 écrit le binaire `rotate` (`cmd/rotate`, cas d'usage dans `internal/app`, adaptateurs IAM, Secret Manager, conteneurs et GitHub dans `internal/adapter`) et ajoute `github.com/scaleway/scaleway-sdk-go` comme dépendance directe, ainsi que `golang.org/x/crypto`. Les versions sont relevées avec `/check-versions` et notées dans `docs/standards.md`.
- #308 :
  - ajoute le job, son application IAM, ses politiques et ses secrets dans OpenTofu, et active les alertes de l'Audit Trail ;
  - retire de `deploy/tofu/storage.tf` les ressources `scaleway_iam_api_key` des buckets, et la variable `key_expiry` ;
  - retire du déploiement l'écriture des clés du bucket, en s'assurant qu'il ne les efface pas. Sinon, un déploiement réécrirait d'anciennes clés.
- Les secrets GitHub `API_SNAPSHOT_BUCKET_*` et `SYNC_SNAPSHOT_BUCKET_*` disparaissent une fois #308 en production.
- Les rotations manuelles de #238 (03/12/2026) et #254 (15/01/2027) restent nécessaires si #308 n'est pas en production à ces dates.
- Cette décision précise l'ADR 0005 (Secrets, Clé de la CI) : la rotation n'y est plus manuelle.
