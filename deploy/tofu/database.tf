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
  # No public endpoint: the migrations run inside the Private Network too
  # (migrate.tf, #259). The instance certificate names the Private Network
  # address only.
}

locals {
  # With IPAM the endpoint has an address and no host name; the instance
  # certificate covers that address (SAN IP Address), so verify-full holds.
  database_private_host = coalesce(scaleway_rdb_instance.main.private_network[0].hostname, scaleway_rdb_instance.main.private_network[0].ip)
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

# The administrator applies the migrations, so it creates the tables: all
# on the database. Admin rights do not override the permissions of a
# logical database, and only the instance's default database gets them
# automatically (Scaleway documentation, managed-databases how-to
# manage-users): without this, its connection is refused (#261).
resource "scaleway_rdb_privilege" "admin" {
  instance_id   = scaleway_rdb_instance.main.id
  database_name = scaleway_rdb_database.api.name
  user_name     = scaleway_rdb_instance.main.user_name
  permission    = "all"
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
