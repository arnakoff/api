# SPDX-License-Identifier: AGPL-3.0-only
#
# Daily backup of the API keys out of Scaleway (#75), with the least rights:
# Scaleway has no permission set narrower than RelationalDatabasesFullAccess
# to create or export a managed backup (iam/policies-permissions/
# permission-sets), so no Scaleway API key takes part. A private container
# on the Private Network runs `admin backup -listen` as arnakoff_backup,
# which may only read the tables (database.tf); a cron trigger posts to
# /backup once a day. The dump is sealed with age for the owner's public
# key, which cannot open it, and added to an Amazon S3 bucket in eu-west-3
# whose key may only add objects (deploy/tofu-aws). The deploy workflow sets
# its image and its secrets, as for the api container.

resource "scaleway_container" "backup" {
  depends_on   = [scaleway_rdb_privilege.backup]
  name         = "backup"
  namespace_id = scaleway_container_namespace.main.id
  image        = var.image
  command      = ["/usr/local/bin/admin"]
  args         = ["backup", "-listen", ":8080"]
  port         = 8080
  protocol     = "http1"

  # Reached by its trigger only, which works with both endpoints off
  # (serverless-containers/how-to/add-trigger-to-a-container).
  privacy                        = "private"
  enable_default_public_endpoint = false
  enable_private_endpoint        = false
  private_network_id             = scaleway_vpc_private_network.main.id

  # Started by the trigger, stopped after: nothing runs between backups.
  cpu_limit          = 250
  memory_limit_bytes = 256000000
  min_scale          = 0
  max_scale          = 1
  # Above the server's write timeout (30 s), which bounds a backup.
  timeout = 60

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

  # Plain values only; the database URL of arnakoff_backup, the log key and
  # the keys of the bucket come from the deploy workflow.
  environment_variables = {
    ARNAKOFF_DATABASE_CA      = scaleway_rdb_instance.main.certificate
    ARNAKOFF_BACKUP_BUCKET    = var.backup_bucket
    ARNAKOFF_BACKUP_RECIPIENT = var.backup_recipient
  }

  lifecycle {
    ignore_changes = [image, secret_environment_variables]
  }
}

resource "scaleway_container_trigger" "backup" {
  container_id = scaleway_container.backup.id
  name         = "backup-daily"
  description  = "Daily backup of the API keys to Amazon S3 (#75)"
  destination_config {
    http_path   = "/backup"
    http_method = "post"
  }
  cron {
    schedule = var.backup_schedule
    timezone = "Europe/Paris"
  }
}
