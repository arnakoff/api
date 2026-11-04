# SPDX-License-Identifier: AGPL-3.0-only
#
# The migrations, applied from inside the Private Network by a container
# that exists only while var.migrate is set (#259): the database never has
# a public endpoint. Serverless Jobs cannot join a Private Network yet
# (Scaleway documentation, serverless-jobs/faq), a Serverless Container can.
# `admin migrate -listen` serves /readyz, 200 once the migrations are
# applied; the owner waits for it, then applies again without var.migrate,
# which removes the container and its secrets and extends the rights of the
# API role to the new tables (database.tf).

resource "scaleway_container" "migrate" {
  count        = var.migrate ? 1 : 0
  depends_on   = [scaleway_rdb_privilege.admin]
  name         = "migrate"
  namespace_id = scaleway_container_namespace.main.id
  image        = var.image
  command      = ["/usr/local/bin/admin"]
  args         = ["migrate", "-listen", ":8080"]
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

  # A migration that fails keeps the process up with /readyz at 503, so
  # that the container shows unready instead of restarting in a loop.
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
    ARNAKOFF_DATABASE_CA = scaleway_rdb_instance.main.certificate
  }
  secret_environment_variables = {
    ARNAKOFF_DATABASE_URL = "postgres://${scaleway_rdb_instance.main.user_name}:${urlencode(var.db_admin_password)}@${local.database_private_host}:${scaleway_rdb_instance.main.private_network[0].port}/${scaleway_rdb_database.api.name}?sslmode=verify-full"
    ARNAKOFF_LOG_KEY      = var.migrate_log_key
  }
}

output "migrate_container_id" {
  description = "UUID of the migrate container while var.migrate is set, for `scw container container get`; empty otherwise."
  value       = var.migrate ? split("/", scaleway_container.migrate[0].id)[1] : ""
}
