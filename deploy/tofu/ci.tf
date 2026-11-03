# SPDX-License-Identifier: AGPL-3.0-only
#
# The CI application `ci-deploy` was created by hand with its first policy
# (ADR 0005). It also writes the versions of the sync job's secrets, which
# ADR 0005 left for #71.

data "scaleway_iam_application" "ci_deploy" {
  name = "ci-deploy"
}

resource "scaleway_iam_policy" "ci_deploy_secrets" {
  name           = "ci-deploy-secrets"
  description    = "CI: write the secret versions of the sync job"
  application_id = data.scaleway_iam_application.ci_deploy.id
  rule {
    project_ids          = [data.scaleway_account_project.current.id]
    permission_set_names = ["SecretManagerSecretWrite"]
  }
}
