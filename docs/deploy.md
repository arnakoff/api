# Déploiement en production

Procédure d'exploitation de l'API chez Scaleway, région fr-par (ADR 0005, #71). Trois acteurs :

- **OpenTofu** (`deploy/tofu/`), lancé par le propriétaire depuis sa machine, crée et règle l'infrastructure : base, bucket, réseau privé, conteneur `api`, job `sync`, clés et politiques IAM. La CI ne le lance jamais.
- **Le workflow Deploy** (`.github/workflows/deploy.yml`), dans l'environnement GitHub `production` qui exige la revue du propriétaire : il vérifie la signature de l'image, la copie dans Container Registry, recopie les secrets de l'environnement et déploie le digest.
- **Le propriétaire**, à la main : les migrations (`admin migrate`), l'enregistrement DNS et la rotation des secrets.

Les valeurs marquées « à vérifier » n'ont pas pu être confirmées dans la documentation ; elles se contrôlent au premier déploiement et ce fichier est corrigé ensuite.

## Avant le premier déploiement

Le code doit contenir #246 (clés par lien de connexion), #247 (`ARNAKOFF_CLIENT_IP_HEADER`) et #249 (`ARNAKOFF_DATABASE_CA`), publiés dans une version : la configuration de `deploy/tofu` leur donne leurs variables.

Outils : OpenTofu 1.13.2, la CLI `scw` avec le profil `arnakoff` de `~/.config/scw/config.yaml` (projet `arnakoff-production`), `gh` connecté au dépôt, `docker`, `jq`, `cosign`. Sous Windows, tout se lance depuis Git Bash, dans un seul terminal gardé ouvert jusqu'au bout.

Dans chaque terminal où OpenTofu tourne :

```sh
export SCW_PROFILE=arnakoff   # sans lui, scw et OpenTofu prennent le profil par défaut
# Le backend S3 lit les clés AWS_* : ce sont celles du profil arnakoff.
export AWS_ACCESS_KEY_ID="$(scw config get access-key | tail -n 1)"
export AWS_SECRET_ACCESS_KEY="$(scw config get secret-key | tail -n 1)"
read -rs TF_VAR_state_passphrase && export TF_VAR_state_passphrase   # 16 caractères au moins, gardée hors du dépôt
# Le propriétaire : seul utilisateur que la politique du bucket des instantanés laisse le gérer.
export TF_VAR_owner_user_id="$(scw iam api-key get "$AWS_ACCESS_KEY_ID" -o json | tail -n 1 | jq -r .APIKey.user_id)"
# Adresse que les alertes de Cockpit préviennent (#72) : donnée personnelle, hors du dépôt.
export TF_VAR_alert_email=<adresse e-mail du propriétaire>
```

`| tail -n 1` : sous Windows, `scw` écrit son avertissement sur les permissions de `config.yaml` sur la sortie standard, avant la valeur, et Windows ne permet pas de le faire taire (relevé le 04/11/2026, `scw` 2.65.2).

### 1. Bucket de l'état OpenTofu (une fois)

```sh
scw object bucket create name=arnakoff-tofu-state enable-versioning=true acl=private region=fr-par
```

Le premier `tofu apply` (étape 4) pose sur ce bucket une politique qui le réserve au propriétaire (`scaleway_object_bucket_policy.state`, `deploy/tofu/storage.tf`) : sans elle, les clés des applications `api` et `sync`, qui ont des droits Object Storage sur tout le projet, pourraient y lire ou écrire (#289).

L'état est chiffré sur la machine avant l'écriture (chiffrement d'état d'OpenTofu, `pbkdf2` et `aes_gcm`, imposé) : sans la phrase de passe, il est illisible et OpenTofu refuse de l'écrire en clair. Perdre la phrase de passe, c'est perdre l'état.

### 2. Secrets générés par le propriétaire (une fois)

`gh secret set` lit la valeur sur l'entrée standard sans l'afficher.

```sh
openssl rand -hex 32 | gh secret set ARNAKOFF_LOG_KEY --env production
openssl rand -hex 32 | gh secret set ARNAKOFF_API_KEY_SECRET --env production
gh secret set ARNAKOFF_SMTP_PASSWORD --env production        # mot de passe de contact@arnakoff.com
# ARNAKOFF_TURNSTILE_SECRET est déjà dans l'environnement.
```

Deux mots de passe de base, choisis maintenant et gardés dans le gestionnaire de mots de passe : `TF_VAR_db_admin_password` (rôle `arnakoff_admin`, migrations) et `TF_VAR_db_api_password` (rôle `arnakoff_api`). Ils sont en écriture seule : OpenTofu les envoie à Scaleway sans les garder dans l'état.

```sh
read -rs TF_VAR_db_admin_password && export TF_VAR_db_admin_password
read -rs TF_VAR_db_api_password && export TF_VAR_db_api_password
```

### 3. Première image dans Container Registry

Le conteneur et le job se créent avec une image ; le workflow Deploy la remplace ensuite.

```sh
scw config get secret-key | tail -n 1 | docker login rg.fr-par.scw.cloud/arnakoff -u nologin --password-stdin
docker buildx imagetools create --tag rg.fr-par.scw.cloud/arnakoff/api:<version> ghcr.io/arnakoff/api@<digest linux/amd64>
```

Le digest `linux/amd64` se lit avec `docker buildx imagetools inspect ghcr.io/arnakoff/api:<version>`, après `cosign verify` (README). `imagetools create` enveloppe ce manifeste dans un nouvel index : le tag copié a un autre digest, le manifeste garde le sien (constaté le 04/11/2026 avec 0.52.0).

### 4. Infrastructure

```sh
cd deploy/tofu
tofu init
tofu apply \
  -var image=rg.fr-par.scw.cloud/arnakoff/api:<version> \
  -var key_expiry=<date RFC 3339 dans la durée maximale des clés de l'organisation> \
  -var api_hostname=   # pas de domaine tant que son CNAME n'existe pas (étape 8)
```

La base n'a alors aucun point d'accès public : seul le réseau privé y mène.

Le conteneur est créé sans ses secrets, qui viennent au déploiement : il reste en erreur (`ARNAKOFF_LOG_KEY: required`…) jusqu'à l'étape 7, c'est attendu (constaté le 04/11/2026). Les secrets du job `sync` reçoivent une version provisoire que le premier déploiement remplace ; d'ici là, une exécution planifiée du job échoue.

### 5. Valeurs de la CI

```sh
gh variable set SCW_CONTAINER_ID --env production --body "$(tofu output -raw container_id)"
gh variable set SCW_CONTAINER_ENDPOINT --env production --body "$(tofu output -raw container_endpoint)"
gh variable set SCW_JOB_DEFINITION_ID --env production --body "$(tofu output -raw job_definition_id)"
gh variable set SCW_SYNC_SECRET_IDS --env production --body "$(tofu output -raw sync_secret_ids)"
tofu output -raw api_bucket_access_key  | gh secret set API_SNAPSHOT_BUCKET_ACCESS_KEY --env production
tofu output -raw api_bucket_secret_key  | gh secret set API_SNAPSHOT_BUCKET_SECRET_KEY --env production
tofu output -raw sync_bucket_access_key | gh secret set SYNC_SNAPSHOT_BUCKET_ACCESS_KEY --env production
tofu output -raw sync_bucket_secret_key | gh secret set SYNC_SNAPSHOT_BUCKET_SECRET_KEY --env production
```

URL de la base, par le réseau privé, avec `verify-full` (le certificat de l'instance est passé au conteneur par `ARNAKOFF_DATABASE_CA`) :

```sh
printf 'postgres://arnakoff_api:%s@%s:%s/arnakoff?sslmode=verify-full' \
  "$(printf %s "$TF_VAR_db_api_password" | jq -sRj @uri)" \
  "$(tofu output -raw database_private_host)" "$(tofu output -raw database_private_port)" \
  | gh secret set ARNAKOFF_DATABASE_URL --env production
```

Avec l'IPAM, la base n'a pas de nom d'hôte sur le réseau privé : `database_private_host` est son adresse. Le certificat de l'instance la couvre (SAN `IP Address`, relevé le 04/11/2026), `verify-full` tient. Si la connexion échoue sur le certificat, ouvrir une issue avant d'affaiblir quoi que ce soit.

### 6. Migrations, puis droits

Les migrations tournent dans le réseau privé, dans le conteneur `migrate` (`deploy/tofu/migrate.tf`, #259) : la base n'a jamais de point d'accès public. Il faut une image qui contient `admin migrate -listen` (la première version publiée avec #259, ou une plus récente), copiée dans Container Registry comme à l'étape 3.

```sh
export TF_VAR_image=rg.fr-par.scw.cloud/arnakoff/api:<version>
export TF_VAR_migrate_log_key="$(openssl rand -hex 32)"
tofu apply -var migrate=true
scw container container get "$(tofu output -raw migrate_container_id)" | grep '^Status'   # ready : migrations appliquées
tofu apply                                                                               # sans migrate
unset TF_VAR_migrate_log_key
```

Le conteneur reçoit l'URL du rôle `arnakoff_admin` vers l'adresse privée de la base, en `verify-full`, et l'autorité de l'instance. Il applique les migrations, puis répond `200` sur `/readyz` : Scaleway le marque alors prêt (`ready`). Un échec laisse `/readyz` à `503` et le conteneur non prêt, sans redémarrer en boucle ; ses journaux se lisent dans Cockpit. Le mot de passe admin est dans l'état, chiffré, tant que le conteneur existe.

Le second `tofu apply` retire le conteneur et ses secrets, et étend le droit `readwrite` du rôle `arnakoff_api` aux tables que la migration vient de créer : Scaleway n'applique un droit qu'aux objets qui existent à ce moment (documentation de `scaleway_rdb_privilege`, « Permission Drift »). « À vérifier » : que `readwrite` donne bien `SELECT`, `INSERT`, `UPDATE` et `DELETE` (la purge en a besoin, ADR 0006).

Le rôle `arnakoff_admin` a la permission `all` sur la base (`scaleway_rdb_privilege.admin`) : les droits d'administrateur de Scaleway ne valent pas permission sur une base logique, et sans elle la connexion est refusée (`permission denied for database "arnakoff"`, constaté le 04/11/2026, #261). Un échec de migration se lit avec `scw container container logs "$(tofu output -raw migrate_container_id)"` ; Scaleway arrête le conteneur resté non prêt au bout de 5 minutes et le relance.

« À vérifier » au premier usage : qu'un conteneur privé sans point d'accès public (`enable_default_public_endpoint = false`) passe bien ses sondes.

La base n'a pas de point d'accès public. Si elle en avait un (un `load_balancer` créé à la main ou par une ancienne configuration), le retirer de la configuration ne le supprime pas : OpenTofu ne détruit que son ACL, ce qui l'ouvrirait à toute adresse (liste par défaut `0.0.0.0/0`). Le supprimer d'abord avec `scw rdb endpoint delete <id du point d'accès> instance-id=<id de l'instance>` (constaté le 04/11/2026).

### 7. Premier déploiement

1. Lancer une première synchronisation, sans laquelle `/readyz` répond 503 et le conteneur n'est jamais prêt : écrire d'abord les secrets du job en lançant le workflow Deploy (il échouera à l'attente du conteneur), puis `scw jobs definition start <SCW_JOB_DEFINITION_ID>` et suivre le run avec `scw jobs run list`.
2. Relancer le workflow Deploy : Actions → Deploy → Run workflow, version `<version>`, puis approuver l'environnement `production`.

Ensuite, chaque version publiée par release-please déclenche Deploy après le workflow Image ; il attend l'approbation du propriétaire.

### 8. Domaine `api.arnakoff.com`

1. Chez Cloudflare : `CNAME api` → `tofu output -raw container_host`, en « DNS only » (le certificat Let's Encrypt est émis par Scaleway).
2. `tofu apply ...`, sans `-var api_hostname=` : `api_hostname` vaut `api.arnakoff.com` par défaut.
3. Mettre `SCW_CONTAINER_ENDPOINT` à `https://api.arnakoff.com`.

Le domaine est protégé par `prevent_destroy` (#273) : un plan qui le détruirait ou le remplacerait échoue (`has prevent_destroy set, but the plan calls for it to be destroyed`) au lieu de le proposer. Ce message veut presque toujours dire qu'une variable manque à la commande. Pour retirer ou changer le domaine volontairement : enlever le bloc `lifecycle` de `scaleway_container_domain.api` dans une PR, appliquer, puis le remettre.

## Observabilité et alertes

Choix du propriétaire (#72, 04/11/2026) : les métriques et les journaux que Cockpit collecte seul, des alertes par e-mail, une sonde externe dans GitHub Actions. L'application n'envoie rien à Cockpit.

- **Métriques et journaux** : Cockpit collecte ceux du conteneur `api`, du job `sync` et de la base, gratuitement dans la durée de conservation par défaut (31 jours pour les métriques, 7 jours pour les journaux ; documentation de Scaleway, `cockpit-product-integration` et `cockpit-pricing`). Ils se lisent dans Grafana : Cockpit → Grafana, tableaux de bord préconfigurés de Serverless Containers, Serverless Jobs et Managed Database. Pour le conteneur : état, nombre d'instances, CPU, mémoire, requêtes par seconde et part de chaque code de statut (`serverless-containers/how-to/monitor-container`) ; pas de latence. La latence se lit dans les journaux d'accès de l'API (Serverless Containers logs), qui donnent la route, le statut et la durée de chaque requête, sans le chemin ni la requête du client.
- **Alertes Cockpit** (`deploy/tofu/observability.tf`) : le gestionnaire d'alertes de fr-par prévient `alert_email`. Alertes préconfigurées activées : échec d'une exécution du job (`JobRunFailed`), et, pour la base, charge CPU, stockage et nombre de connexions (`PostgreSQLHighCPULoad`, `PostgreSQLHighStorageUsage`, `PostgresqlTooManyConnections`). `sync -once` sort en erreur dès qu'une source échoue ou que son instantané est rejeté : chaque échec de synchronisation est donc un e-mail. Cockpit n'a pas d'alerte préconfigurée pour Serverless Containers (liste de l'API Cockpit du projet, relevée le 04/11/2026). Après le premier `tofu apply` : Cockpit → Alertes → Contacts → « Send test alert ».
- **Sonde externe** (`.github/workflows/probe.yml`) : toutes les 15 minutes, `GET /readyz` doit répondre 200, et chaque source téléchargée de `GET /v1/sources` doit être `ok` avec des données de moins de 24 heures (quatre synchronisations manquées). Un échec est l'alerte : GitHub prévient par e-mail, selon ses réglages de notification Actions, l'utilisateur qui a créé le workflow ou modifié son `cron` en dernier. GitHub peut retarder une exécution planifiée, et désactive la planification d'un dépôt public après 60 jours sans activité : la réactiver dans Actions → Probe. Lancement à la main : Actions → Probe → Run workflow.
- **Webhook `ARNAKOFF_ALERT_WEBHOOK`** : reste disponible pour un environnement local ou un autre hébergement ; il n'est pas réglé en production.

## Registre MCP

Le serveur est publié dans le registre MCP officiel sous le nom `com.arnakoff/api` (#66), prouvé par l'enregistrement TXT `v=MCPv1; k=ed25519; p=<CLÉ_PUBLIQUE>` à l'apex de `arnakoff.com`, dans la zone Cloudflare. Le workflow `.github/workflows/mcp-registry.yml` publie `server.json` à chaque tag `vX.Y.Z` de release-please (#278) ; release-please y met la version.

Avant le premier tag qui le lance, le propriétaire crée l'environnement (Settings → Environments → New environment) :

- nom `mcp-registry` ; sans lui, GitHub le crée au premier lancement, sans aucune protection ;
- la règle de tag ne protège que parce que le jeu de règles « version tags » réserve la création, le déplacement et la suppression des tags `v*` à l'application de release : il doit le rester ;
- « Deployment branches and tags » : « Selected branches and tags », règle de tag `v*` ;
- secret d'environnement `MCP_PRIVATE_KEY` : la clé privée en hexadécimal (64 caractères), jamais un secret de dépôt :

```sh
openssl pkey -in ~/.arnakoff/mcp-registry-key.pem -noout -text | grep -A3 "priv:" | tail -n +2 | tr -d ' :\n' | gh secret set MCP_PRIVATE_KEY --env mcp-registry
```

- relecteur obligatoire (« Required reviewers »), au choix du propriétaire : chaque publication attend alors son accord.

Le registre garde chaque version publiée et refuse de la modifier : le job s'arrête avant la connexion si la version de `server.json` n'est pas celle du tag, et relancer le job d'une version déjà publiée échoue. Publication à la main : `mcp-publisher login dns`, puis `mcp-publisher publish` (#66).

## Opérations courantes

- **Retour arrière** : Actions → Deploy → Run workflow avec la version précédente. Une migration n'est jamais annulée par un retour arrière : une migration doit rester compatible avec la version précédente du code.
- **Nouvelle migration** : avant le déploiement de la version qui l'apporte, refaire l'étape 6 avec cette version (`migrate=true`, attendre `ready`, puis `tofu apply` sans `migrate`, qui retire le conteneur et étend les droits).
- **Rotation** :
  - clés du bucket : nouvelle `key_expiry`, `tofu apply` (les clés sont recréées), puis l'étape 5 pour les clés et un déploiement ;
  - mots de passe de base : incrémenter `db_api_password_version` (ou `db_admin_password_version`) avec le nouveau mot de passe, `tofu apply`, nouvelle `ARNAKOFF_DATABASE_URL`, déploiement ;
  - clé de la CI `ci-deploy` : 30 jours, manuelle (#238).
  - clé du registre MCP : nouvelle paire Ed25519, nouveau TXT à l'apex de `arnakoff.com` (Cloudflare) **et suppression de l'ancien**, que le registre essaie en premier, puis nouveau secret `MCP_PRIVATE_KEY`.
- **Secrets du job `sync`** : réécrits à chaque déploiement dans Secret Manager (`disable-previous=true`).
