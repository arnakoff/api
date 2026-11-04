# SPDX-License-Identifier: AGPL-3.0-only
#
# Snapshot bucket (ADR 0005): sync writes, api reads, each with a key of its
# own application, restricted by the bucket policy.

data "scaleway_account_project" "current" {
  name = "arnakoff-production"
}

resource "scaleway_object_bucket" "snapshots" {
  name = "arnakoff-snapshots"

  versioning {
    enabled = true
  }

  # Replaced snapshots stay 30 days, to restore one that slipped through.
  lifecycle_rule {
    id      = "expire-replaced-snapshots"
    enabled = true
    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

resource "scaleway_iam_application" "api" {
  name        = "arnakoff-api"
  description = "api container: reads the snapshot bucket"
}

resource "scaleway_iam_application" "sync" {
  name        = "arnakoff-sync"
  description = "sync job: reads and writes the snapshot bucket"
}

resource "scaleway_iam_policy" "api" {
  name           = "arnakoff-api-snapshots"
  application_id = scaleway_iam_application.api.id
  rule {
    project_ids          = [data.scaleway_account_project.current.id]
    permission_set_names = ["ObjectStorageBucketsRead", "ObjectStorageObjectsRead"]
  }
}

resource "scaleway_iam_policy" "sync" {
  name           = "arnakoff-sync-snapshots"
  application_id = scaleway_iam_application.sync.id
  rule {
    project_ids          = [data.scaleway_account_project.current.id]
    permission_set_names = ["ObjectStorageBucketsRead", "ObjectStorageObjectsRead", "ObjectStorageObjectsWrite"]
  }
}

resource "scaleway_iam_api_key" "api" {
  application_id     = scaleway_iam_application.api.id
  description        = "api: snapshot bucket, read only"
  default_project_id = data.scaleway_account_project.current.id
  expires_at         = var.key_expiry
}

resource "scaleway_iam_api_key" "sync" {
  application_id     = scaleway_iam_application.sync.id
  description        = "sync: snapshot bucket, read and write"
  default_project_id = data.scaleway_account_project.current.id
  expires_at         = var.key_expiry
}

# A bucket policy replaces the default access of the project, so the owner
# keeps full access by name. Version 2023-04-17 only accepts user_id,
# application_id or "*" as principals; project_id belongs to the deprecated
# 2012-10-17 (Scaleway documentation, object-storage/api-cli/bucket-policy).
# The Owner of the Organization may always edit the policy itself.
resource "scaleway_object_bucket_policy" "snapshots" {
  bucket = scaleway_object_bucket.snapshots.id
  policy = jsonencode({
    Version = "2023-04-17"
    Statement = [
      {
        Sid       = "Owner keeps full access"
        Effect    = "Allow"
        Principal = { SCW = "user_id:${var.owner_user_id}" }
        Action    = ["*"]
        Resource  = [scaleway_object_bucket.snapshots.name, "${scaleway_object_bucket.snapshots.name}/*"]
      },
      {
        Sid       = "api reads the snapshots"
        Effect    = "Allow"
        Principal = { SCW = "application_id:${scaleway_iam_application.api.id}" }
        Action    = ["s3:ListBucket", "s3:GetObject"]
        Resource  = [scaleway_object_bucket.snapshots.name, "${scaleway_object_bucket.snapshots.name}/*"]
      },
      {
        Sid       = "sync writes the snapshots"
        Effect    = "Allow"
        Principal = { SCW = "application_id:${scaleway_iam_application.sync.id}" }
        Action    = ["s3:ListBucket", "s3:GetObject", "s3:PutObject"]
        Resource  = [scaleway_object_bucket.snapshots.name, "${scaleway_object_bucket.snapshots.name}/*"]
      },
    ]
  })
}

# The OpenTofu state bucket, created by hand before the first apply
# (docs/deploy.md), holds the encrypted state and its lock. The api and
# sync keys have Object Storage rights on the whole project: without a
# policy of its own, the bucket would fall back to them (#289). This policy
# leaves it to the owner, who applies this configuration with their key.
resource "scaleway_object_bucket_policy" "state" {
  bucket = local.state_bucket
  policy = jsonencode({
    Version = "2023-04-17"
    Statement = [
      {
        Sid       = "Owner only"
        Effect    = "Allow"
        Principal = { SCW = "user_id:${var.owner_user_id}" }
        Action    = ["*"]
        Resource  = [local.state_bucket, "${local.state_bucket}/*"]
      },
    ]
  })
}

locals {
  # The backend block of versions.tf cannot read a local: keep both equal.
  state_bucket = "arnakoff-tofu-state"
}
