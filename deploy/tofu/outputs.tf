# SPDX-License-Identifier: AGPL-3.0-only
#
# What the deploy workflow and the owner need after an apply (docs/deploy.md).

# Resource IDs carry their region ("fr-par/<uuid>"); the scw CLI only
# accepts the UUID (`scw container container get fr-par/<uuid>`: Not Found,
# scw 2.65.2, 04/11/2026).
output "container_id" {
  description = "GitHub variable SCW_CONTAINER_ID of the environment production: the container's UUID."
  value       = split("/", scaleway_container.api.id)[1]
}

# public_endpoint, with or without its scheme: the documentation does not
# say, so both forms are normalised.
locals {
  container_host = trimsuffix(trimprefix(scaleway_container.api.public_endpoint, "https://"), "/")
}

output "container_endpoint" {
  description = "GitHub variable SCW_CONTAINER_ENDPOINT: URL of the container's default endpoint."
  value       = "https://${local.container_host}"
}

output "container_host" {
  description = "Host of the container's default endpoint: the target of the CNAME of api_hostname."
  value       = local.container_host
}

output "backup_container_id" {
  description = "GitHub variable SCW_BACKUP_CONTAINER_ID of the environment production: the backup container's UUID."
  value       = split("/", scaleway_container.backup.id)[1]
}

output "job_definition_id" {
  description = "GitHub variable SCW_JOB_DEFINITION_ID of the environment production: the job definition's UUID."
  value       = split("/", scaleway_job_definition.sync.id)[1]
}

output "rotate_job_definition_id" {
  description = "GitHub variable SCW_ROTATE_JOB_DEFINITION_ID of the environment production: the rotate job definition's UUID."
  value       = split("/", scaleway_job_definition.rotate.id)[1]
}

output "rotate_application_id" {
  description = "IAM application of the rotate job, whose first key the owner creates at the bootstrap (docs/deploy.md)."
  value       = scaleway_iam_application.rotate.id
}

output "rotate_secret_ids" {
  description = "Secret Manager IDs of the rotate job, by variable, for its bootstrap (docs/deploy.md)."
  value       = jsonencode(local.rotate_secret_ids)
}

output "sync_secret_ids" {
  description = "GitHub variable SCW_SYNC_SECRET_IDS: JSON object of the Secret Manager IDs of the sync job, by variable."
  value       = jsonencode({ for k, s in scaleway_secret.sync : k => split("/", s.id)[1] })
}

output "database_private_host" {
  description = "Host of the database on the Private Network, for ARNAKOFF_DATABASE_URL: its host name, or else its address."
  value       = local.database_private_host
}

output "database_private_port" {
  description = "Port of the database on the Private Network."
  value       = scaleway_rdb_instance.main.private_network[0].port
}


output "database_certificate" {
  description = "CA certificate of the database, for sslrootcert with sslmode=verify-full."
  value       = scaleway_rdb_instance.main.certificate
}
