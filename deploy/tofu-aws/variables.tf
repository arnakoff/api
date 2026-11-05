# SPDX-License-Identifier: AGPL-3.0-only

variable "state_passphrase" {
  description = "Passphrase of the state encryption, at least 16 characters, kept by the owner (TF_VAR_state_passphrase)."
  type        = string
  sensitive   = true
}

variable "aws_account_id" {
  description = "ID of the account arnakoff-production, the only one this configuration may change: `aws sts get-caller-identity --profile arnakoff-production --query Account`."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.aws_account_id))
    error_message = "aws_account_id must be a 12-digit AWS account ID."
  }
}

variable "bucket_name" {
  description = "Name of the bucket of the backups, unique across Amazon S3."
  type        = string
  default     = "arnakoff-key-backups"
}

variable "retention_days" {
  description = "Days a backup is kept, decided by the owner (#75, ADR 0006); it is deleted for good one day later."
  type        = number
  default     = 30
}

# The subject of the OIDC token of a job on main. The repository was created
# on 29/10/2026, after 15/07/2026, so the claim names the immutable IDs of
# its owner and itself (GitHub documentation, configuring OpenID Connect in
# Amazon Web Services); they are `gh api repos/arnakoff/api --jq
# '.owner.id, .id'`.
variable "github_subject" {
  description = "sub claim of the GitHub job allowed to list the backups: the Backup age workflow on main."
  type        = string
  default     = "repo:arnakoff@349736528/api@1438163844:ref:refs/heads/main"
}
