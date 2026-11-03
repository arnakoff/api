# SPDX-License-Identifier: AGPL-3.0-only
#
# Managed PostgreSQL of the API keys (ADR 0004, ADR 0005).

resource "scaleway_vpc_private_network" "main" {
  name = "arnakoff"
}

resource "scaleway_rdb_instance" "main" {
  name               = "arnakoff"
  node_type          = "DB-DEV-S"
  engine             = "PostgreSQL-17"
  is_ha_cluster      = false
  encryption_at_rest = true

  # The administrator applies the migrations (ADR 0004); its password is
  # write-only, never in the state.
  user_name           = "arnakoff_admin"
  password_wo         = var.db_admin_password
  password_wo_version = var.db_admin_password_version

  # Daily backup kept 7 days (ADR 0005); the copy out of Scaleway is #75.
  disable_backup            = false
  backup_schedule_frequency = 24
  backup_schedule_retention = 7

  # The API reaches the database through the Private Network only.
  private_network {
    pn_id       = scaleway_vpc_private_network.main.id
    enable_ipam = true
  }

  # Public endpoint, for `admin migrate` from the owner's machine only, and
  # only while var.migration_cidrs is set: its default ACL is 0.0.0.0/0, any
  # host (Scaleway, manage-allowed-ip-addresses), so it never exists without
  # the ACL below. ACLs apply to the public endpoint only, never to the
  # Private Network.
  dynamic "load_balancer" {
    for_each = length(var.migration_cidrs) > 0 ? [true] : []
    content {}
  }
}

resource "scaleway_rdb_acl" "migrations" {
  count       = length(var.migration_cidrs) > 0 ? 1 : 0
  instance_id = scaleway_rdb_instance.main.id

  dynamic "acl_rules" {
    for_each = var.migration_cidrs
    content {
      ip          = acl_rules.value
      description = "admin migrate from the owner"
    }
  }
}

resource "scaleway_rdb_database" "api" {
  instance_id = scaleway_rdb_instance.main.id
  name        = "arnakoff"
}

resource "scaleway_rdb_user" "api" {
  instance_id         = scaleway_rdb_instance.main.id
  name                = "arnakoff_api"
  password_wo         = var.db_api_password
  password_wo_version = var.db_api_password_version
  is_admin            = false
}

# readwrite on the tables that exist when it is applied: after a migration
# that adds a table, `tofu apply` again to extend it (permission drift,
# scaleway_rdb_privilege documentation; docs/deploy.md).
resource "scaleway_rdb_privilege" "api" {
  instance_id   = scaleway_rdb_instance.main.id
  database_name = scaleway_rdb_database.api.name
  user_name     = scaleway_rdb_user.api.name
  permission    = "readwrite"
}
