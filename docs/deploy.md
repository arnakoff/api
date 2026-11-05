# Déploiement en production

Procédure d'exploitation de l'API chez Scaleway, région fr-par (ADR 0005, #71). Trois acteurs :

- **OpenTofu** (`deploy/tofu/`, et `deploy/tofu-aws/` pour le bucket des sauvegardes chez Amazon S3), lancé par le propriétaire depuis sa machine, crée et règle l'infrastructure : base, bucket, réseau privé, conteneurs `api` et `backup`, job `sync`, clés et politiques IAM. La CI ne le lance jamais.
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

## Sauvegarde des clés hors de Scaleway (#75)

Choix du propriétaire (05/11/2026) : une copie quotidienne de la table `api_keys`, chiffrée, dans un bucket Amazon S3 à Paris (`eu-west-3`), gardée 30 jours, avec le moins de droits possible.

- **Pourquoi pas la sauvegarde gérée** : Scaleway crée ses sauvegardes automatiques au format instantané (`managed-databases-for-postgresql-and-mysql/how-to/manage-snapshots`), et créer ou exporter une sauvegarde logique demande `RelationalDatabasesFullAccess`, qui permet aussi de supprimer la base : il n'existe rien entre `RelationalDatabasesReadOnly` et `FullAccess` (`iam/policies-permissions/permission-sets`). Aucune clé d'API Scaleway ne sert donc à la sauvegarde.
- **Comment** : le conteneur privé `backup` (`deploy/tofu/backup.tf`), sur le réseau privé, sans point d'accès, démarre au déclencheur cron de 03:40 (Europe/Paris), qui poste sur `/backup`, puis s'arrête (`min_scale = 0`). Il lance `admin backup -listen` avec le rôle `arnakoff_backup`, qui ne peut que lire (`readonly`). Le dump (archive tar : `manifest.json` et `api_keys.csv`, lus dans une seule transaction en lecture seule) est chiffré avec age pour la clé publique du propriétaire, puis ajouté sous `keys/<date UTC>.tar.age`. Les demandes de lien de connexion n'y sont pas copiées.
- **Droits côté AWS** (`deploy/tofu-aws`) : la clé du conteneur ne peut que `s3:PutObject` sous `keys/`, et seulement avec `If-None-Match: *` ; elle ne lit, ne liste, ne remplace ni ne supprime rien. Le workflow `Backup age` liste le bucket avec le jeton OIDC de son job sur `main`, sans clé stockée, et ne peut lire aucune sauvegarde. Bucket privé, versionné, chiffré côté serveur, TLS imposé.
- **Durée** : une sauvegarde expire au bout de 30 jours et sa version est supprimée le lendemain (ADR 0006).
- **Surveillance** : `.github/workflows/backup-age.yml` échoue chaque matin si la sauvegarde la plus récente a plus de 26 heures, ou s'il n'y en a aucune ; GitHub prévient par e-mail comme pour la sonde. Un échec du conteneur se lit dans Cockpit (journaux du conteneur `backup`, message `backup failed`).
- **Données personnelles** : la copie contient les adresses e-mail des titulaires de compte. Amazon n'en reçoit que le chiffré ; la clé privée age ne quitte jamais le propriétaire. Ce transfert à un sous-traitant américain, même en région Paris, est à inscrire au registre des traitements (#74), avec sa base légale (certification d'AWS au EU-US Data Privacy Framework « à vérifier »).

### Mise en place (une fois)

1. **Clé age**, sur la machine du propriétaire, avec age 1.3.2 ou plus récent :

   ```sh
   age-keygen -pq -o ~/.arnakoff/backup-age-key.txt   # clé privée : gestionnaire de mots de passe et copie hors ligne
   age-keygen -y ~/.arnakoff/backup-age-key.txt       # clé publique age1pq1…, TF_VAR_backup_recipient
   ```

   Perdre la clé privée, c'est perdre toutes les sauvegardes.

2. **Compte AWS** : l'organisation AWS `arnakoff` a deux comptes, `arnakoff-production`, qui porte le bucket, et `arnakoff-val`, que cette configuration ne touche pas. Le propriétaire s'y connecte par IAM Identity Center, avec l'accès administrateur sur `arnakoff-production`, sans clé d'accès longue durée. Profil SSO `arnakoff-production` dans `~/.aws/config`, créé une fois :

   ```sh
   aws configure sso --profile arnakoff-production   # SSO region : celle d'IAM Identity Center ; compte arnakoff-production, rôle administrateur ; Default client Region : eu-west-3
   aws sso login --profile arnakoff-production       # avant chaque session : les identifiants SSO expirent
   aws sts get-caller-identity --profile arnakoff-production --query Account --output text   # TF_VAR_aws_account_id
   ```

   `allowed_account_ids` fait échouer OpenTofu avant tout changement si le profil pointe sur un autre compte. Les clés Scaleway de l'état vont dans le profil `scaleway-arnakoff` de `~/.aws/credentials`, pour que le fournisseur AWS ne les prenne jamais :

   ```ini
   [scaleway-arnakoff]
   aws_access_key_id = <access key Scaleway du profil arnakoff>
   aws_secret_access_key = <secret key Scaleway du profil arnakoff>
   ```

3. **Bucket et identités AWS**, dans un terminal **sans** `AWS_ACCESS_KEY_ID` ni `AWS_SECRET_ACCESS_KEY` :

   ```sh
   unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY
   cd deploy/tofu-aws
   read -rs TF_VAR_state_passphrase && export TF_VAR_state_passphrase
   export TF_VAR_aws_account_id=<ID du compte arnakoff-production>
   aws sso login --profile arnakoff-production
   tofu init && tofu apply
   gh variable set AWS_BACKUP_BUCKET --body "$(tofu output -raw backup_bucket)"
   gh variable set AWS_BACKUP_CHECK_ROLE_ARN --body "$(tofu output -raw backup_check_role_arn)"
   tofu output -raw backup_access_key | gh secret set BACKUP_BUCKET_ACCESS_KEY --env production
   tofu output -raw backup_secret_key | gh secret set BACKUP_BUCKET_SECRET_KEY --env production
   ```

   Les deux variables sont des variables de dépôt : le workflow `Backup age` ne tourne pas dans l'environnement `production`. Le `sub` attendu (`github_subject`) porte les identifiants immuables du dépôt, créé après le 15/07/2026 : vérifié le 05/11/2026, le rôle accepte le jeton du job sur `main`. Un refus `Not authorized to perform sts:AssumeRoleWithWebIdentity` voudrait dire que le `sub` a changé.

4. **Conteneur et rôle**, dans `deploy/tofu`, avec les variables habituelles de l'étape 4 et :

   ```sh
   read -rs TF_VAR_db_backup_password && export TF_VAR_db_backup_password   # sans espace, gestionnaire de mots de passe
   export TF_VAR_backup_bucket=arnakoff-key-backups
   export TF_VAR_backup_recipient="$(age-keygen -y ~/.arnakoff/backup-age-key.txt)"   # clé publique age1pq1…
   tofu apply
   gh variable set SCW_BACKUP_CONTAINER_ID --env production --body "$(tofu output -raw backup_container_id)"
   printf 'postgres://arnakoff_backup:%s@%s:%s/arnakoff?sslmode=verify-full' \
     "$(printf %s "$TF_VAR_db_backup_password" | jq -sRj @uri)" \
     "$(tofu output -raw database_private_host)" "$(tofu output -raw database_private_port)" \
     | gh secret set BACKUP_DATABASE_URL --env production
   ```

   `TF_VAR_image` doit contenir `admin backup` : la version 0.57.0 ou une plus récente, copiée dans Container Registry par un premier déploiement.

5. **Déploiement** : un lancement de Deploy après l'étape 4 (Actions → Deploy → Run workflow) donne au conteneur son image et ses secrets ; avant `SCW_BACKUP_CONTAINER_ID`, l'étape du conteneur `backup` est sautée. La première sauvegarde a lieu à 03:40 ; le matin même, Actions → Backup age doit passer. Vérifié le 05/11/2026 : le conteneur privé à `min_scale = 0` démarre à son déclencheur et la sauvegarde arrive dans le bucket dans la seconde (`keys/2026-11-05T090500Z.tar.age`, 4 731 octets).

   Pour un essai sans attendre la nuit : `tofu apply -var backup_schedule="<minute> <heure> * * *"` (heure de Paris), puis `tofu apply` sans cette variable pour revenir à 03:40.

### Restauration

`admin restore` charge un dump dans une base **vide** : il migre le schéma jusqu'à la version du dump, charge les lignes en une seule transaction, recale la séquence des identifiants, puis applique les migrations suivantes ; il refuse une base qui contient déjà des clés ou un schéma plus récent que le dump.

**Exercice, sur la machine du propriétaire**, après la mise en place puis une fois par trimestre :

```sh
aws sso login --profile arnakoff-production
aws s3 cp "s3://arnakoff-key-backups/keys/<date>.tar.age" . --profile arnakoff-production
age -d -i ~/.arnakoff/backup-age-key.txt -o keys.tar "<date>.tar.age"
docker run -d --name restore -e POSTGRES_PASSWORD=test -p 127.0.0.1:5433:5432 postgres:17.11
export ARNAKOFF_LOG_KEY="$(openssl rand -hex 32)" ARNAKOFF_API_KEY_SECRET="$(openssl rand -hex 32)"
export ARNAKOFF_DATABASE_URL='postgres://postgres:test@127.0.0.1:5433/postgres?sslmode=disable'
go run ./cmd/admin restore -file keys.tar && go run ./cmd/admin key list
docker rm -f restore && rm keys.tar "<date>.tar.age"
```

`key list` doit montrer les clés de la production à la date du dump. `keys.tar` est en clair : il contient des adresses e-mail, ne quitte pas la machine et se supprime aussitôt.

**En production** :

- base perdue ou abîmée depuis moins de 7 jours : la sauvegarde gérée de Scaleway suffit (console → Managed Database → Backups → Restore), sans la copie AWS ;
- Scaleway perdu, ou sauvegarde gérée inutilisable : nouvelle base, puis `admin restore -file keys.tar` avec le rôle des migrations, lancé d'un hôte qui joint la base. La base de production n'ayant pas de point d'accès public, il faut un hôte sur le réseau privé : procédure à écrire et à exercer (#297). `ARNAKOFF_API_KEY_SECRET` n'est pas dans la sauvegarde (ADR 0004) : sans lui, les empreintes restaurées ne vérifient plus aucune clé ; il se garde aussi dans le gestionnaire de mots de passe du propriétaire.

## Observabilité et alertes

Choix du propriétaire (#72, 04/11/2026) : les métriques et les journaux que Cockpit collecte seul, des alertes par e-mail, une sonde externe dans GitHub Actions. L'application n'envoie rien à Cockpit.

- **Métriques et journaux** : Cockpit collecte ceux du conteneur `api`, du job `sync` et de la base, gratuitement dans la durée de conservation par défaut (31 jours pour les métriques, 7 jours pour les journaux ; documentation de Scaleway, `cockpit-product-integration` et `cockpit-pricing`). Ils se lisent dans Grafana : Cockpit → Grafana, tableaux de bord préconfigurés de Serverless Containers, Serverless Jobs et Managed Database. Pour le conteneur : état, nombre d'instances, CPU, mémoire, requêtes par seconde et part de chaque code de statut (`serverless-containers/how-to/monitor-container`) ; pas de latence. La latence se lit dans les journaux d'accès de l'API (Serverless Containers logs), qui donnent la route, le statut et la durée de chaque requête, sans le chemin ni la requête du client.
- **Alertes Cockpit** (`deploy/tofu/observability.tf`) : le gestionnaire d'alertes de fr-par prévient `alert_email`. Alertes préconfigurées activées : échec d'une exécution du job (`JobRunFailed`), et, pour la base, charge CPU, stockage et nombre de connexions (`PostgreSQLHighCPULoad`, `PostgreSQLHighStorageUsage`, `PostgresqlTooManyConnections`). `sync -once` sort en erreur dès qu'une source échoue ou que son instantané est rejeté : chaque échec de synchronisation est donc un e-mail. Cockpit n'a pas d'alerte préconfigurée pour Serverless Containers (liste de l'API Cockpit du projet, relevée le 04/11/2026) : les conteneurs ont les alertes sur les journaux ci-dessous. Après le premier `tofu apply` : Cockpit → Alertes → Contacts → « Send test alert ».
- **Alertes sur les journaux** (`deploy/cockpit/logs-alerts.yaml`, #270) : deux règles évaluées par la source `Scaleway Logs` du projet, chaque minute, qui préviennent le même gestionnaire d'alertes :
  - `ApiHigh5xxRate` : plus de 5 % des requêtes du conteneur `api` des 10 dernières minutes en 5xx, sur au moins 30 requêtes (0,05 par seconde), seuils choisis par le propriétaire le 05/11/2026 ;
  - `BackupFailed` : le conteneur `backup` a journalisé `backup failed` (réponse 500 à `POST /backup`) ou `cannot start` dans les 15 dernières minutes, sans attendre `Backup age` le lendemain matin.

  Elles comptent les lignes de journal : la métrique `serverless_container_requests_per_second` vaut 0 au trafic actuel (relevé du 05/11/2026), et sa part de 5xx serait toujours vide. Le fournisseur OpenTofu n'a pas de ressource pour ces règles : le propriétaire les envoie avec sa clé Scaleway, à chaque changement du fichier (API Loki de Cockpit, `cockpit/api-cli/create-alerts-programmatically`) :

  ```sh
  export SCW_PROFILE=arnakoff
  LOGS="$(scw cockpit data-source list -o json | tail -n 1 | jq -r '.[] | select(.name == "Scaleway Logs") | .url')"
  curl -fsS -X POST -H "X-Auth-Token: $(scw config get secret-key | tail -n 1)" -H "Content-Type: application/yaml" \
    --data-binary @deploy/cockpit/logs-alerts.yaml "$LOGS/loki/api/v1/rules/arnakoff"
  curl -fsS -H "X-Auth-Token: $(scw config get secret-key | tail -n 1)" "$LOGS/loki/api/v1/rules/arnakoff"   # relire les règles envoyées
  ```

  Coût d'une règle d'alerte personnalisée « à vérifier » : la page de tarification de Cockpit n'en parle pas (`cockpit/reference-content/cockpit-pricing`, relevé du 05/11/2026).
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
- **Nouvelle migration** : avant le déploiement de la version qui l'apporte, refaire l'étape 6 avec cette version (`migrate=true`, attendre `ready`, puis `tofu apply` sans `migrate`, qui retire le conteneur et étend les droits, `readonly` du rôle `arnakoff_backup` compris). Une migration qui ajoute une table dit si la sauvegarde la copie (`backedUp` ou `leftOut`, `internal/adapter/store/postgres/backup.go`) : un test l'exige.
- **Rotation** :
  - clés du bucket : nouvelle `key_expiry`, `tofu apply` (les clés sont recréées), puis l'étape 5 pour les clés et un déploiement ;
  - mots de passe de base : incrémenter `db_api_password_version` (ou `db_admin_password_version`, `db_backup_password_version`) avec le nouveau mot de passe, `tofu apply`, nouvelle `ARNAKOFF_DATABASE_URL` (ou `BACKUP_DATABASE_URL`), déploiement ;
  - clé AWS du conteneur `backup` : `tofu apply -replace=aws_iam_access_key.backup` dans `deploy/tofu-aws`, nouveaux `BACKUP_BUCKET_ACCESS_KEY` et `BACKUP_BUCKET_SECRET_KEY`, déploiement ;
  - clé de la CI `ci-deploy` : 30 jours, manuelle (#238).
  - clé du registre MCP : nouvelle paire Ed25519, nouveau TXT à l'apex de `arnakoff.com` (Cloudflare) **et suppression de l'ancien**, que le registre essaie en premier, puis nouveau secret `MCP_PRIVATE_KEY`.
- **Secrets du job `sync`** : réécrits à chaque déploiement dans Secret Manager (`disable-previous=true`).
