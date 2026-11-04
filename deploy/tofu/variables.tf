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

variable "image" {
  description = "Image the container and the job are created with, in Container Registry. The deploy workflow replaces it afterwards."
  type        = string
}

variable "api_hostname" {
  description = "Custom domain of the API, such as api.arnakoff.com, once its CNAME points at the container's default endpoint. Empty: none."
  type        = string
  default     = ""
}

variable "key_expiry" {
  description = "Expiry date of the bucket keys of api and sync, RFC 3339, within the organisation's maximum key lifetime. Changing it recreates the keys."
  type        = string
}

variable "sync_schedule" {
  description = "Cron schedule of the sync job, Europe/Paris time: the most frequent source is synced every 6 hours (internal/adapter/source/catalog)."
  type        = string
  default     = "17 */6 * * *"
}

variable "demo_origin" {
  description = "Origin of the site, the only one CORS allows on the demo and key request routes."
  type        = string
  default     = "https://arnakoff.com"
}
