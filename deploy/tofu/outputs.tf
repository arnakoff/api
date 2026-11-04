# SPDX-License-Identifier: AGPL-3.0-only
#
# What the deploy workflow and the owner need after an apply (docs/deploy.md).
# The keys are sensitive: read them with `tofu output -raw` piped into
# `gh secret set`, never printed.

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

output "job_definition_id" {
  description = "GitHub variable SCW_JOB_DEFINITION_ID of the environment production: the job definition's UUID."
  value       = split("/", scaleway_job_definition.sync.id)[1]
}

output "sync_secret_ids" {
  description = "GitHub variable SCW_SYNC_SECRET_IDS: JSON object of the Secret Manager IDs of the sync job, by variable."
  value       = jsonencode({ for k, s in scaleway_secret.sync : k => split("/", s.id)[1] })
}

# With IPAM, the endpoint on the Private Network has an address and no host
# name (`scw rdb instance get`: "name": null). The instance certificate
# covers that address (SAN IP Address), so verify-full holds.
output "database_private_host" {
  description = "Host of the database on the Private Network, for ARNAKOFF_DATABASE_URL: its host name, or else its address."
  value       = coalesce(scaleway_rdb_instance.main.private_network[0].hostname, scaleway_rdb_instance.main.private_network[0].ip)
}

output "database_private_port" {
  description = "Port of the database on the Private Network."
  value       = scaleway_rdb_instance.main.private_network[0].port
}

output "database_public_endpoint" {
  description = "Public endpoint of the database, host:port, for `admin migrate` from var.migration_cidrs; empty while it is closed."
  value = try(
    "${scaleway_rdb_instance.main.load_balancer[0].hostname}:${scaleway_rdb_instance.main.load_balancer[0].port}", "",
  )
}

output "database_certificate" {
  description = "CA certificate of the database, for sslrootcert with sslmode=verify-full."
  value       = scaleway_rdb_instance.main.certificate
}

output "api_bucket_access_key" {
  value     = scaleway_iam_api_key.api.access_key
  sensitive = true
}

output "api_bucket_secret_key" {
  value     = scaleway_iam_api_key.api.secret_key
  sensitive = true
}

output "sync_bucket_access_key" {
  value     = scaleway_iam_api_key.sync.access_key
  sensitive = true
}

output "sync_bucket_secret_key" {
  value     = scaleway_iam_api_key.sync.secret_key
  sensitive = true
}
