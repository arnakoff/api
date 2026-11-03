# SPDX-License-Identifier: AGPL-3.0-only

variable "state_passphrase" {
  description = "Passphrase of the state encryption, at least 16 characters, kept by the owner (TF_VAR_state_passphrase)."
  type        = string
  sensitive   = true
}

variable "db_admin_password" {
  description = "Password of the database administrator, the migration role (ADR 0004). Write-only: never stored in the state."
  type        = string
  sensitive   = true
  ephemeral   = true
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

variable "migration_cidrs" {
  description = "Addresses allowed on the public endpoint of the database, to run `admin migrate`: the owner's, as CIDR. Empty: no public endpoint at all."
  type        = list(string)
  default     = []
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
