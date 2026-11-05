# SPDX-License-Identifier: AGPL-3.0-only
#
# The key rotation job (ADR 0007, #308): every day it replaces the Scaleway
# keys of arnakoff-api, arnakoff-sync, ci-deploy and its own once one is
# older than 30 days (cmd/rotate). OpenTofu creates no key: the job owns
# them all, and a key in this state would be recreated by the next apply
# once the job had deleted it. Its first key is written by hand once
# (docs/deploy.md, « Rotation automatique des clés »).

resource "scaleway_iam_application" "rotate" {
  name        = "arnakoff-rotate"
  description = "rotate job: replaces the keys of api, sync, ci-deploy and its own (ADR 0007)"
}

# The exception to least privilege that ADR 0007 accepts: no narrower right
# than IAMApplicationManager manages the keys of an application, and it is
# scoped to the organization. The job writes the secrets of the api
# container (ContainersFullAccess, no narrower write right) and new
# versions of the Secret Manager secrets of sync and of itself: Create makes
# the version, Write disables the previous one (#264). It reads no secret
# through the API, its own come from the job definition: no
# SecretManagerSecretAccess.
resource "scaleway_iam_policy" "rotate" {
  name           = "arnakoff-rotate"
  description    = "rotate job: keys of the applications, secrets of api and sync (ADR 0007)"
  application_id = scaleway_iam_application.rotate.id
  rule {
    organization_id      = data.scaleway_account_project.current.organization_id
    permission_set_names = ["IAMApplicationManager"]
  }
  rule {
    project_ids          = [data.scaleway_account_project.current.id]
    permission_set_names = ["ContainersFullAccess", "SecretManagerSecretCreate", "SecretManagerSecretWrite"]
  }
}

# Secrets only the job reads: its own key, one secret per half, which it
# rewrites at each rotation, and the private key of the GitHub App, written
# by the owner. ARNAKOFF_LOG_KEY is the one of the sync job.
resource "scaleway_secret" "rotate" {
  for_each = toset([
    "ARNAKOFF_ROTATE_ACCESS_KEY", "ARNAKOFF_ROTATE_SECRET_KEY", "ARNAKOFF_GITHUB_APP_PRIVATE_KEY",
  ])
  name        = "rotate-${lower(replace(each.key, "_", "-"))}"
  description = "${each.key} of the rotate job, written at the bootstrap (docs/deploy.md), then by the job for its key"
}

# As for the sync job: a placeholder version, write-only, until the owner
# writes the real ones at the bootstrap; until then, a run fails.
resource "scaleway_secret_version" "rotate_placeholder" {
  for_each        = scaleway_secret.rotate
  secret_id       = each.value.id
  description     = "Placeholder until the bootstrap"
  data_wo         = "unset-until-the-bootstrap"
  data_wo_version = 1
}

locals {
  # Secret Manager and container IDs carry their region ("fr-par/<uuid>");
  # cmd/rotate takes the UUID.
  rotate_secret_ids = { for k, s in scaleway_secret.rotate : k => split("/", s.id)[1] }
  sync_secret_ids   = { for k, s in scaleway_secret.sync : k => split("/", s.id)[1] }

  rotate_secrets = merge(
    { for k, s in scaleway_secret.rotate : k => s.id },
    { ARNAKOFF_LOG_KEY = scaleway_secret.sync["ARNAKOFF_LOG_KEY"].id },
  )
}

resource "scaleway_job_definition" "rotate" {
  name      = "rotate"
  image_uri = var.image
  cpu_limit = 250
  # In MB and MiB (scaleway_job_definition documentation).
  memory_limit           = 512
  local_storage_capacity = 1024
  # A rotation day takes about 20 minutes at worst: up to 5 minutes for a
  # new bucket key to work, twice, and 10 for the api container.
  timeout = "45m"

  startup_command = ["/usr/local/bin/rotate"]

  # 03:10, out of the sync runs (sync_schedule, 30 minutes at most).
  cron {
    schedule = "10 3 * * *"
    timezone = "Europe/Paris"
  }

  env = {
    ARNAKOFF_ROTATE_ORGANIZATION_ID       = data.scaleway_account_project.current.organization_id
    ARNAKOFF_ROTATE_PROJECT_ID            = data.scaleway_account_project.current.id
    ARNAKOFF_ROTATE_BUCKET                = scaleway_object_bucket.snapshots.name
    ARNAKOFF_ROTATE_APP_API               = scaleway_iam_application.api.id
    ARNAKOFF_ROTATE_APP_SYNC              = scaleway_iam_application.sync.id
    ARNAKOFF_ROTATE_APP_CI                = data.scaleway_iam_application.ci_deploy.id
    ARNAKOFF_ROTATE_APP_SELF              = scaleway_iam_application.rotate.id
    ARNAKOFF_ROTATE_API_CONTAINER_ID      = split("/", scaleway_container.api.id)[1]
    ARNAKOFF_ROTATE_SYNC_ACCESS_SECRET_ID = local.sync_secret_ids["ARNAKOFF_SNAPSHOT_BUCKET_ACCESS_KEY"]
    ARNAKOFF_ROTATE_SYNC_SECRET_SECRET_ID = local.sync_secret_ids["ARNAKOFF_SNAPSHOT_BUCKET_SECRET_KEY"]
    ARNAKOFF_ROTATE_SELF_ACCESS_SECRET_ID = local.rotate_secret_ids["ARNAKOFF_ROTATE_ACCESS_KEY"]
    ARNAKOFF_ROTATE_SELF_SECRET_SECRET_ID = local.rotate_secret_ids["ARNAKOFF_ROTATE_SECRET_KEY"]
    ARNAKOFF_GITHUB_APP_CLIENT_ID         = var.github_app_client_id
    ARNAKOFF_GITHUB_APP_INSTALLATION_ID   = var.github_app_installation_id
  }

  # Version "latest" by default: each run reads the last version, the one
  # the previous run or the bootstrap wrote, as the sync job does.
  dynamic "secret_reference" {
    for_each = local.rotate_secrets
    content {
      secret_id   = secret_reference.value
      environment = secret_reference.key
    }
  }

  depends_on = [scaleway_secret_version.rotate_placeholder]

  lifecycle {
    ignore_changes = [image_uri]
  }
}

# The bucket keys belong to the rotate job now: forgotten by OpenTofu, kept
# in Scaleway until the job replaces them (#308).
removed {
  from = scaleway_iam_api_key.api
  lifecycle {
    destroy = false
  }
}

removed {
  from = scaleway_iam_api_key.sync
  lifecycle {
    destroy = false
  }
}
