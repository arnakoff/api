# SPDX-License-Identifier: AGPL-3.0-only
#
# What deploy/tofu and the GitHub environment production need
# (docs/deploy.md). The keys are sensitive: read them with
# `tofu output -raw` piped into `gh secret set`, never printed.

output "backup_bucket" {
  description = "var.backup_bucket of deploy/tofu, and GitHub variable AWS_BACKUP_BUCKET."
  value       = aws_s3_bucket.backups.bucket
}

output "backup_access_key" {
  description = "GitHub secret BACKUP_BUCKET_ACCESS_KEY of the environment production."
  value       = aws_iam_access_key.backup.id
  sensitive   = true
}

output "backup_secret_key" {
  description = "GitHub secret BACKUP_BUCKET_SECRET_KEY of the environment production."
  value       = aws_iam_access_key.backup.secret
  sensitive   = true
}

output "backup_check_role_arn" {
  description = "GitHub variable AWS_BACKUP_CHECK_ROLE_ARN: the role the Backup age workflow assumes."
  value       = aws_iam_role.backup_check.arn
}
