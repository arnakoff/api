# SPDX-License-Identifier: AGPL-3.0-only
#
# Alerts of the production (#72). Cockpit collects the metrics and logs of
# Serverless Containers, Serverless Jobs and Managed Database by itself, for
# free within the default retention (scaleway/docs-content, pages/cockpit/
# reference-content/cockpit-product-integration.mdx and cockpit-pricing.mdx):
# the application pushes nothing. Its alert manager notifies by email only
# (pages/cockpit/how-to/enable-alert-manager.mdx), the channel the owner
# chose. Availability is probed from outside by .github/workflows/probe.yml.

# Preconfigured alerts enabled, by rule name, as listed by the Cockpit API
# for this project on 04/11/2026 (GET /cockpit/v1/regions/fr-par/alerts,
# is_preconfigured=true). Serverless Containers have none.
locals {
  preconfigured_alerts = toset([
    # sync -once exits non-zero when a source fails or is rejected, which
    # fails the run: the sync failure alert.
    "JobRunFailed",
    "PostgreSQLHighCPULoad",
    "PostgreSQLHighStorageUsage",
    "PostgresqlTooManyConnections",
  ])
  preconfigured_alert_ids = [
    for a in data.scaleway_cockpit_preconfigured_alert.all.alerts :
    a.preconfigured_rule_id if contains(local.preconfigured_alerts, a.name)
  ]
}

data "scaleway_cockpit_preconfigured_alert" "all" {
  project_id = data.scaleway_account_project.current.id
}

resource "scaleway_cockpit_alert_manager" "main" {
  project_id              = data.scaleway_account_project.current.id
  preconfigured_alert_ids = local.preconfigured_alert_ids

  contact_points {
    email = var.alert_email
  }

  lifecycle {
    precondition {
      condition     = length(local.preconfigured_alert_ids) == length(local.preconfigured_alerts)
      error_message = "A preconfigured alert of local.preconfigured_alerts is not offered by Cockpit any more: check the names with the scaleway_cockpit_preconfigured_alert data source."
    }
  }
}
