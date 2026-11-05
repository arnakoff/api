# SPDX-License-Identifier: AGPL-3.0-only
#
# The mails of the API, the login links of the free keys (#59), go through
# Scaleway Transactional Email (#295): Serverless Containers refuse
# outbound traffic on ports 25 and 465 except to TEM (scaleway/docs-content,
# pages/serverless-containers/reference-content/containers-limitations.mdx,
# "Blocked ports"), so the mailbox of OVH cannot be used to send.
#
# The domain was created and its SPF and DKIM records written in Cloudflare
# by hand on 05/11/2026: it is imported, never created, since a new domain
# would have another DKIM key. Its name, accept_tos and project are
# ForceNew in the provider (v2.89.0): prevent_destroy refuses any plan that
# would replace it. The MX stay those of OVH, where contact@arnakoff.com
# receives.
import {
  to = scaleway_tem_domain.main
  id = "fr-par/1ed8ba71-da5b-4189-b03b-8c989a914a7d"
}

resource "scaleway_tem_domain" "main" {
  project_id = data.scaleway_account_project.current.id
  name       = "arnakoff.com"
  accept_tos = true

  lifecycle {
    prevent_destroy = true
  }
}

# The api container authenticates as the project with a key of this
# application, which may only send mail by SMTP. The rotate job creates and
# replaces its key and writes it to ARNAKOFF_SMTP_PASSWORD (ADR 0007).
resource "scaleway_iam_application" "mail" {
  name        = "arnakoff-mail"
  description = "api container: sends the login links through Transactional Email by SMTP"
}

resource "scaleway_iam_policy" "mail" {
  name           = "arnakoff-mail-smtp"
  description    = "api container: sends mail by SMTP only, no domain configuration (#295)"
  application_id = scaleway_iam_application.mail.id
  rule {
    project_ids          = [data.scaleway_account_project.current.id]
    permission_set_names = ["TransactionalEmailEmailSmtpCreate"]
  }
}

locals {
  # Implicit TLS (RFC 8314), the only mode of internal/adapter/smtpmail;
  # the username is the project of the domain.
  smtp_addr     = "${scaleway_tem_domain.main.smtp_host}:${scaleway_tem_domain.main.smtps_port}"
  smtp_username = scaleway_tem_domain.main.smtps_auth_user
}

output "tem_dns_records" {
  description = "Records the TEM domain expects in the Cloudflare zone of arnakoff.com, to compare with the zone by hand."
  value = {
    spf_value  = scaleway_tem_domain.main.spf_value
    dkim_name  = scaleway_tem_domain.main.dkim_name
    dkim_value = scaleway_tem_domain.main.dkim_config
    status     = scaleway_tem_domain.main.status
  }
}
