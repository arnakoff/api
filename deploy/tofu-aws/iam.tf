# SPDX-License-Identifier: AGPL-3.0-only
#
# Two identities, each with one right (#75):
# - the backup container adds objects under keys/, with a static key, as
#   Scaleway cannot authenticate to AWS with OIDC. It cannot read, list,
#   replace (If-None-Match: *) nor delete a backup;
# - the Backup age workflow lists the names and dates of the backups, with
#   the OIDC token of its job on main and no stored key. It cannot read
#   one.

resource "aws_iam_user" "backup" {
  name = "arnakoff-backup"
}

resource "aws_iam_user_policy" "backup" {
  name = "add-backups"
  user = aws_iam_user.backup.name
  policy = jsonencode({
    Version = "2012-10-17"
    # Only with If-None-Match, so that the key cannot replace a backup with
    # a new version, which the lifecycle would delete a day later (Amazon
    # S3 User Guide, conditional-writes-enforce, example 1). Without
    # multipart uploads: a backup is one PutObject.
    Statement = [{
      Effect    = "Allow"
      Action    = "s3:PutObject"
      Resource  = "${aws_s3_bucket.backups.arn}/keys/*"
      Condition = { Null = { "s3:if-none-match" = "false" } }
    }]
  })
}

# Its secret is in the state, encrypted. Rotation: docs/deploy.md.
resource "aws_iam_access_key" "backup" {
  user = aws_iam_user.backup.name
}

# AWS checks the certificate of GitHub against its own trusted CAs: no
# thumbprint (aws_iam_openid_connect_provider documentation, 6.71.0).
resource "aws_iam_openid_connect_provider" "github" {
  url            = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]
}

resource "aws_iam_role" "backup_check" {
  name                 = "arnakoff-backup-check"
  max_session_duration = 3600
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.github.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          "token.actions.githubusercontent.com:sub" = var.github_subject
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "backup_check" {
  name = "list-backups"
  role = aws_iam_role.backup_check.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Action    = "s3:ListBucket"
      Resource  = aws_s3_bucket.backups.arn
      Condition = { StringEquals = { "s3:prefix" = "keys/" } }
    }]
  })
}
