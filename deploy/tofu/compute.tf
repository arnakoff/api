# SPDX-License-Identifier: AGPL-3.0-only
#
# The api container and the sync job, on the same image (ADR 0005). OpenTofu
# sets their shape and their plain variables; the deploy workflow sets the
# image and every secret from the GitHub environment `production` (ADR 0005,
# Secrets), so both are ignored here once created.

resource "scaleway_container_namespace" "main" {
  name        = "arnakoff"
  description = "arnakoff/api"
}

locals {
  # The most api instances at once, also told to the API: each keeps to its
  # share of the registries' limits per IP address (#336).
  api_max_scale = 2
}

resource "scaleway_container" "api" {
  name         = "api"
  namespace_id = scaleway_container_namespace.main.id
  image        = var.image
  port         = 8100
  protocol     = "http1"
  privacy      = "public"

  # 250 mvCPU with 512 MB (ADR 0005). At least one instance stays up: the
  # snapshots and the per-minute limits live in its memory.
  cpu_limit          = 250
  memory_limit_bytes = 512000000
  min_scale          = 1
  max_scale          = local.api_max_scale
  timeout            = 30

  https_connections_only         = true
  enable_default_public_endpoint = true
  # Off: a caller on the Private Network could set X-Real-IP itself, which
  # the API trusts as the client's address (#247).
  enable_private_endpoint = false
  private_network_id      = scaleway_vpc_private_network.main.id

  # Ready once every snapshot is loaded (ADR 0005), alive while it answers.
  startup_probe {
    failure_threshold = 30
    interval          = "10s"
    timeout           = "5s"
    http {
      path = "/readyz"
    }
  }
  liveness_probe {
    failure_threshold = 3
    interval          = "30s"
    timeout           = "5s"
    http {
      path = "/healthz"
    }
  }

  # Plain values only: the keys of the bucket are Secrets of the API
  # (internal/platform/config) and come with the other secrets. The CA of
  # the instance is public (#249); ARNAKOFF_CLIENT_IP_HEADER is #247.
  environment_variables = {
    ARNAKOFF_API_KEY_ENV      = "live"
    ARNAKOFF_CLIENT_IP_HEADER = "X-Real-IP"
    ARNAKOFF_DATABASE_CA      = scaleway_rdb_instance.main.certificate
    ARNAKOFF_DEMO_ORIGIN      = var.demo_origin
    ARNAKOFF_INSTANCES        = tostring(local.api_max_scale)
    ARNAKOFF_SNAPSHOT_BUCKET  = scaleway_object_bucket.snapshots.name
    ARNAKOFF_SMTP_ADDR        = local.smtp_addr
    ARNAKOFF_SMTP_USERNAME    = local.smtp_username
  }

  lifecycle {
    ignore_changes = [image, secret_environment_variables]
  }
}

# The public address of the API in production. An apply that forgot
# api_hostname once planned to destroy it (#273): prevent_destroy makes such
# a plan fail instead, including when count drops to 0 (checked with
# OpenTofu 1.13.2). Removing the domain on purpose: docs/deploy.md.
resource "scaleway_container_domain" "api" {
  count        = var.api_hostname == "" ? 0 : 1
  container_id = scaleway_container.api.id
  hostname     = var.api_hostname

  lifecycle {
    prevent_destroy = true
  }
}

# Secrets of the sync job, mounted from Secret Manager (ADR 0005). Their
# versions are written by the deploy workflow. ARNAKOFF_ALERT_WEBHOOK is not
# one: the sync alerts reach the owner by the Cockpit email of a failed run
# (observability.tf, #72).
resource "scaleway_secret" "sync" {
  for_each = toset([
    "ARNAKOFF_LOG_KEY", "ARNAKOFF_SNAPSHOT_BUCKET_ACCESS_KEY", "ARNAKOFF_SNAPSHOT_BUCKET_SECRET_KEY",
  ])
  name        = "sync-${lower(replace(each.key, "_", "-"))}"
  description = "${each.key} of the sync job, written by the deploy workflow"
}

# The job may only reference a secret that has a version, and the deploy
# workflow writes the real ones after this configuration is applied: each
# secret starts with a placeholder version, write-only so that it never
# reaches the state, which the first deploy disables (disable-previous).
# Until then, a scheduled run fails on the placeholder configuration.
resource "scaleway_secret_version" "sync_placeholder" {
  for_each        = scaleway_secret.sync
  secret_id       = each.value.id
  description     = "Placeholder until the first deploy"
  data_wo         = "unset-until-the-first-deploy"
  data_wo_version = 1
}

resource "scaleway_job_definition" "sync" {
  name      = "sync"
  image_uri = var.image
  cpu_limit = 250
  # In MB and MiB (scaleway_job_definition documentation).
  memory_limit           = 512
  local_storage_capacity = 1024
  timeout                = "30m"

  startup_command = ["/usr/local/bin/sync"]
  args            = ["-once"]

  cron {
    schedule = var.sync_schedule
    timezone = "Europe/Paris"
  }

  env = {
    ARNAKOFF_SNAPSHOT_BUCKET = scaleway_object_bucket.snapshots.name
  }

  dynamic "secret_reference" {
    for_each = scaleway_secret.sync
    content {
      secret_id   = secret_reference.value.id
      environment = secret_reference.key
    }
  }

  depends_on = [scaleway_secret_version.sync_placeholder]

  lifecycle {
    ignore_changes = [image_uri]
  }
}
