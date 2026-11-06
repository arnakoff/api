# SPDX-License-Identifier: AGPL-3.0-only

variable "state_passphrase" {
  description = "Passphrase of the state encryption, at least 16 characters, kept by the owner (TF_VAR_state_passphrase)."
  type        = string
  sensitive   = true
}

variable "owner_user_id" {
  description = "IAM user ID of the owner, who applies this configuration: the only user the snapshot bucket policy lets manage the bucket. The UserID of `scw iam api-key get <access key>`."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.owner_user_id))
    error_message = "owner_user_id must be a Scaleway user ID, a lower-case UUID."
  }
}

# Not ephemeral: while var.migrate is set, the migrate container gets it in
# its database URL, so it is in the state, encrypted on the owner's machine,
# until the next apply without var.migrate removes the container (#259).
# The instance itself takes it write-only.
variable "db_admin_password" {
  description = "Password of the database administrator, the migration role (ADR 0004). Without spaces: it goes into a URL."
  type        = string
  sensitive   = true

  validation {
    condition     = !strcontains(var.db_admin_password, " ")
    error_message = "db_admin_password must not contain spaces."
  }
}

variable "db_admin_password_version" {
  description = "Increment to send a new db_admin_password."
  type        = number
  default     = 1
}

variable "db_api_password" {
  description = "Password of the role of the API, which only reads and writes its tables (ADR 0004). Write-only: never stored in the state."
  type        = string
  sensitive   = true
  ephemeral   = true
}

variable "db_api_password_version" {
  description = "Increment to send a new db_api_password."
  type        = number
  default     = 1
}

variable "db_backup_password" {
  description = "Password of arnakoff_backup, the role of the backup container, which may only read the tables (#75). Write-only: never stored in the state."
  type        = string
  sensitive   = true
  ephemeral   = true
}

variable "db_backup_password_version" {
  description = "Increment to send a new db_backup_password."
  type        = number
  default     = 1
}

variable "migrate" {
  description = "Run the migrate container, which applies the migrations from the Private Network with image (docs/deploy.md). Apply without it afterwards: the container is removed."
  type        = bool
  default     = false
}

variable "migrate_log_key" {
  description = "ARNAKOFF_LOG_KEY of the migrate container, 64 hex characters (openssl rand -hex 32); required with migrate."
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = !var.migrate || can(regex("^[0-9a-f]{64}$", var.migrate_log_key))
    error_message = "migrate_log_key must be 64 lower-case hex characters when migrate is set."
  }
}

# Not ephemeral, as db_admin_password: the restore container gets them,
# so they stay in the state history. Each is worthless once the restore is
# over: the presigned URL expires within the hour and the age key opens
# only the object the owner deletes after the restore (docs/deploy.md).
variable "restore" {
  description = "Run the restore container, which loads a dump of the API keys from the Private Network with image (docs/deploy.md, #297). Apply without it afterwards: the container is removed."
  type        = bool
  default     = false
}

variable "restore_database" {
  description = "Database the restore container loads the dump into: arnakoff, the API's, or another name for a drill database, created while restore is set."
  type        = string
  default     = "arnakoff"

  validation {
    condition     = can(regex("^[a-z][a-z0-9_]{0,62}$", var.restore_database))
    error_message = "restore_database must be a lower-case database name."
  }
}

variable "restore_log_key" {
  description = "ARNAKOFF_LOG_KEY of the restore container, 64 hex characters (openssl rand -hex 32); required with restore."
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = !var.restore || can(regex("^[0-9a-f]{64}$", var.restore_log_key))
    error_message = "restore_log_key must be 64 lower-case hex characters when restore is set."
  }
}

variable "restore_url" {
  description = "Presigned URL of the dump sealed for restore_identity, from `aws s3 presign`; required with restore."
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = !var.restore || startswith(var.restore_url, "https://")
    error_message = "restore_url must be the https:// URL printed by aws s3 presign when restore is set."
  }
}

variable "restore_identity" {
  description = "age private key drawn for this restore only (age-keygen -pq), the one the dump at restore_url is sealed for; required with restore."
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = !var.restore || startswith(var.restore_identity, "AGE-SECRET-KEY-")
    error_message = "restore_identity must be an age private key when restore is set."
  }
}

variable "erase" {
  description = "Run the erase container, which deletes the API keys and login requests of the account of erase_email from the Private Network with image (docs/deploy.md, #304). Apply without it afterwards: the container is removed."
  type        = bool
  default     = false
}

variable "erase_email" {
  description = "Address of the account to erase, as the holder gives it, whatever its case; required with erase. Personal data: kept in the encrypted state history only."
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = !var.erase || strcontains(var.erase_email, "@")
    error_message = "erase_email must be an email address when erase is set."
  }
}

variable "erase_log_key" {
  description = "ARNAKOFF_LOG_KEY of the erase container, 64 hex characters (openssl rand -hex 32); required with erase."
  type        = string
  sensitive   = true
  default     = ""

  validation {
    condition     = !var.erase || can(regex("^[0-9a-f]{64}$", var.erase_log_key))
    error_message = "erase_log_key must be 64 lower-case hex characters when erase is set."
  }
}

variable "image" {
  description = "Image the container and the job are created with, in Container Registry. The deploy workflow replaces it afterwards."
  type        = string
}

variable "rotate_image" {
  description = "Image of the rotate job in its own registry, pinned by digest: rg.fr-par.scw.cloud/arnakoff-rotate/api@sha256:…, copied by the owner from a signed release (docs/deploy.md). ci-deploy cannot push there (#340). Empty: no job yet, the first apply creates its project and registry."
  type        = string
  default     = ""

  validation {
    condition     = var.rotate_image == "" || can(regex("^rg[.]fr-par[.]scw[.]cloud/arnakoff-rotate/api@sha256:[0-9a-f]{64}$", var.rotate_image))
    error_message = "rotate_image must be rg.fr-par.scw.cloud/arnakoff-rotate/api@sha256:<64 hex digits>, pinned by digest."
  }
}

variable "api_hostname" {
  description = "Custom domain of the API, once its CNAME points at the container's default endpoint. Empty: none, only before that CNAME exists (docs/deploy.md, steps 4 and 8)."
  type        = string
  default     = "api.arnakoff.com"
}

# The GitHub App that writes the key of ci-deploy to the environment
# production (ADR 0007), created by the owner: not secrets.
variable "github_app_client_id" {
  description = "Client ID of the GitHub App of the rotate job (App settings, « Client ID »)."
  type        = string

  validation {
    condition     = length(var.github_app_client_id) > 0
    error_message = "github_app_client_id is required: create the GitHub App first (docs/deploy.md)."
  }
}

variable "github_app_installation_id" {
  description = "ID of the installation of the GitHub App on arnakoff/api: the number at the end of its configuration URL."
  type        = string

  validation {
    condition     = can(regex("^[1-9][0-9]*$", var.github_app_installation_id))
    error_message = "github_app_installation_id must be a positive integer."
  }
}

variable "sync_schedule" {
  description = "Cron schedule of the sync job, Europe/Paris time: the most frequent source is synced every 6 hours (internal/adapter/source/catalog)."
  type        = string
  default     = "17 */6 * * *"
}

# Personal data of the owner: kept in their tfvars, never committed.
variable "alert_email" {
  description = "Email address the Cockpit alert manager notifies (#72)."
  type        = string
  sensitive   = true

  validation {
    condition     = can(regex("^[^@\\s]+@[^@\\s]+$", var.alert_email))
    error_message = "alert_email must be an email address."
  }
}

variable "demo_origin" {
  description = "Origin of the site, the only one CORS allows on the demo and key request routes."
  type        = string
  default     = "https://arnakoff.com"
}

variable "backup_bucket" {
  description = "Amazon S3 bucket of the backups, the backup_bucket output of deploy/tofu-aws (#75)."
  type        = string
}

variable "backup_recipient" {
  description = "age public key the backups are sealed for: the owner's, from age-keygen -pq (docs/deploy.md). The private key never leaves the owner."
  type        = string

  validation {
    condition     = can(regex("^age1[0-9a-z]+$", var.backup_recipient))
    error_message = "backup_recipient must be an age public key, age1…, never the private AGE-SECRET-KEY-…."
  }
}

variable "backup_schedule" {
  description = "Cron schedule of the backup, Europe/Paris time (#75)."
  type        = string
  default     = "40 3 * * *"
}
