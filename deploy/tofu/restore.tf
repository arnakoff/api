# SPDX-License-Identifier: AGPL-3.0-only
#
# The restore of the API keys from their copy out of Scaleway (#297), from
# inside the Private Network by a container that exists only while
# var.restore is set, as the migrations (migrate.tf): the database never
# has a public endpoint. `admin restore -listen` fetches the dump by a
# presigned URL of Amazon S3, so no AWS key takes part, opens it with an
# age key the owner drew for this restore only, never the key of the
# backups, keeps the clear dump in memory, and serves /readyz, 200 once the
# dump is restored. The owner then applies again without var.restore, which
# removes the container, its secrets and the drill database, if any.

locals {
  # The database restored into: the API's, or a drill database next to it.
  restore_drill    = var.restore && var.restore_database != scaleway_rdb_database.api.name
  restore_database = local.restore_drill ? scaleway_rdb_database.drill[0].name : scaleway_rdb_database.api.name
}

# A database to drill the restore on, beside the API's (docs/deploy.md).
resource "scaleway_rdb_database" "drill" {
  count       = local.restore_drill ? 1 : 0
  instance_id = scaleway_rdb_instance.main.id
  name        = var.restore_database
}

# As scaleway_rdb_privilege.admin: without it, the administrator cannot
# connect to a logical database it did not get by default (#261).
resource "scaleway_rdb_privilege" "drill" {
  count         = local.restore_drill ? 1 : 0
  instance_id   = scaleway_rdb_instance.main.id
  database_name = scaleway_rdb_database.drill[0].name
  user_name     = scaleway_rdb_instance.main.user_name
  permission    = "all"
}

resource "scaleway_container" "restore" {
  count        = var.restore ? 1 : 0
  depends_on   = [scaleway_rdb_privilege.admin, scaleway_rdb_privilege.drill]
  name         = "restore"
  namespace_id = scaleway_container_namespace.main.id
  image        = var.image
  command      = ["/usr/local/bin/admin"]
  args         = ["restore", "-listen", ":8080"]
  port         = 8080
  protocol     = "http1"

  # Nobody calls it: private, without its public endpoint, on the Private
  # Network to reach the database.
  privacy                        = "private"
  enable_default_public_endpoint = false
  enable_private_endpoint        = false
  private_network_id             = scaleway_vpc_private_network.main.id

  cpu_limit          = 250
  memory_limit_bytes = 256000000
  min_scale          = 1
  max_scale          = 1
  timeout            = 30

  # A restore that fails keeps the process up with /readyz at 503, so that
  # the container shows unready instead of restarting in a loop.
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

  environment_variables = {
    ARNAKOFF_DATABASE_CA        = scaleway_rdb_instance.main.certificate
    ARNAKOFF_RESTORE_VERIFY_KEY = var.backup_verify_key
  }
  secret_environment_variables = {
    ARNAKOFF_DATABASE_URL     = "${local.admin_database_server}/${local.restore_database}?sslmode=verify-full"
    ARNAKOFF_LOG_KEY          = var.restore_log_key
    ARNAKOFF_RESTORE_URL      = var.restore_url
    ARNAKOFF_RESTORE_IDENTITY = var.restore_identity
  }
}

output "restore_container_id" {
  description = "UUID of the restore container while var.restore is set, for `scw container container get`; empty otherwise."
  value       = var.restore ? split("/", scaleway_container.restore[0].id)[1] : ""
}
