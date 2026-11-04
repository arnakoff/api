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
```

`| tail -n 1` : sous Windows, `scw` écrit son avertissement sur les permissions de `config.yaml` sur la sortie standard, avant la valeur, et Windows ne permet pas de le faire taire (relevé le 04/11/2026, `scw` 2.65.2).

### 1. Bucket de l'état OpenTofu (une fois)

```sh
scw object bucket create name=arnakoff-tofu-state enable-versioning=true acl=private region=fr-par
```

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
  -var key_expiry=<date RFC 3339 dans la durée maximale des clés de l'organisation>
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

« À vérifier » : que le certificat de l'instance couvre le nom d'hôte du réseau privé. La documentation de Scaleway ne cite que `rw-<id>.rdb.fr-par.scw.cloud`. Si la connexion échoue sur le nom d'hôte, ouvrir une issue avant d'affaiblir quoi que ce soit.

### 6. Migrations, puis droits

Ouvrir le point d'accès public pour sa seule adresse :

```sh
tofu apply -var ... -var 'migration_cidrs=["<votre adresse IPv4>/32"]'
```

Le point d'accès et sa liste d'adresses autorisées sont créés dans le même `apply`, mais l'un après l'autre : pendant ces quelques secondes, la liste par défaut de Scaleway (`0.0.0.0/0`) s'applique. La base reste protégée par TLS et par son mot de passe.

```sh
export ARNAKOFF_DATABASE_URL="postgres://arnakoff_admin:$(printf %s "$TF_VAR_db_admin_password" | jq -sRj @uri)@$(tofu output -raw database_public_endpoint)/arnakoff?sslmode=verify-full"
export ARNAKOFF_DATABASE_CA="$(tofu output -raw database_certificate)"
export ARNAKOFF_LOG_KEY="$(openssl rand -hex 32)"   # journal local seulement
go run ./cmd/admin migrate
tofu apply -var ...   # mêmes variables qu'à l'étape 4, sans migration_cidrs
```

Ce dernier `tofu apply` ferme le point d'accès public et étend le droit `readwrite` du rôle `arnakoff_api` aux tables que la migration vient de créer : Scaleway n'applique un droit qu'aux objets qui existent à ce moment (documentation de `scaleway_rdb_privilege`, « Permission Drift »). « À vérifier » : que `readwrite` donne bien `SELECT`, `INSERT`, `UPDATE` et `DELETE` (la purge en a besoin, ADR 0006).

`admin` lit toute la configuration : laisser `ARNAKOFF_SMTP_ADDR` vide dans ce terminal (sinon il exige aussi le mot de passe SMTP).

### 7. Premier déploiement

1. Lancer une première synchronisation, sans laquelle `/readyz` répond 503 et le conteneur n'est jamais prêt : écrire d'abord les secrets du job en lançant le workflow Deploy (il échouera à l'attente du conteneur), puis `scw jobs definition start <SCW_JOB_DEFINITION_ID>` et suivre le run avec `scw jobs run list`.
2. Relancer le workflow Deploy : Actions → Deploy → Run workflow, version `<version>`, puis approuver l'environnement `production`.

Ensuite, chaque version publiée par release-please déclenche Deploy après le workflow Image ; il attend l'approbation du propriétaire.

### 8. Domaine `api.arnakoff.com`

1. Chez Cloudflare : `CNAME api` → `tofu output -raw container_host`, en « DNS only » (le certificat Let's Encrypt est émis par Scaleway).
2. `tofu apply -var api_hostname=api.arnakoff.com ...`
3. Mettre `SCW_CONTAINER_ENDPOINT` à `https://api.arnakoff.com`.

## Opérations courantes

- **Retour arrière** : Actions → Deploy → Run workflow avec la version précédente. Une migration n'est jamais annulée par un retour arrière : une migration doit rester compatible avec la version précédente du code.
- **Nouvelle migration** : avant le déploiement de la version qui l'apporte, refaire l'étape 6 (ouvrir, migrer, puis `tofu apply` sans `migration_cidrs`, qui ferme le point d'accès et étend les droits).
- **Rotation** :
  - clés du bucket : nouvelle `key_expiry`, `tofu apply` (les clés sont recréées), puis l'étape 5 pour les clés et un déploiement ;
  - mots de passe de base : incrémenter `db_api_password_version` (ou `db_admin_password_version`) avec le nouveau mot de passe, `tofu apply`, nouvelle `ARNAKOFF_DATABASE_URL`, déploiement ;
  - clé de la CI `ci-deploy` : 30 jours, manuelle (#238).
- **Secrets du job `sync`** : réécrits à chaque déploiement dans Secret Manager (`disable-previous=true`).
