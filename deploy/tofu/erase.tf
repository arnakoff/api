# SPDX-License-Identifier: AGPL-3.0-only
#
# The erasure of an account on request (RGPD, article 17, #304), from
# inside the Private Network by a container that exists only while
# var.erase is set, as the restore (restore.tf): the database never has a
# public endpoint. `admin account erase -listen` deletes the keys and the
# login requests of the account of var.erase_email in one transaction and
# serves /readyz, 200 once it is erased. The owner then applies again
# without var.erase, which removes the container and its secrets.

resource "scaleway_container" "erase" {
  count        = var.erase ? 1 : 0
  depends_on   = [scaleway_rdb_privilege.admin]
  name         = "erase"
  namespace_id = scaleway_container_namespace.main.id
  image        = var.image
  command      = ["/usr/local/bin/admin"]
  args         = ["account", "erase", "-listen", ":8080"]
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

  # An erasure that fails keeps the process up with /readyz at 503, so
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
    ARNAKOFF_DATABASE_URL = "${local.admin_database_server}/${scaleway_rdb_database.api.name}?sslmode=verify-full"
    ARNAKOFF_LOG_KEY      = var.erase_log_key
    ARNAKOFF_ERASE_EMAIL  = var.erase_email
  }
}

output "erase_container_id" {
  description = "UUID of the erase container while var.erase is set, for `scw container container get`; empty otherwise."
  value       = var.erase ? split("/", scaleway_container.erase[0].id)[1] : ""
}
