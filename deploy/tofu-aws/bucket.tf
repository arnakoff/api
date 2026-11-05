# SPDX-License-Identifier: AGPL-3.0-only
#
# The backups, sealed with age before they leave Scaleway: Amazon only ever
# holds ciphertext.

resource "aws_s3_bucket" "backups" {
  bucket = var.bucket_name

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_s3_bucket_public_access_block" "backups" {
  bucket                  = aws_s3_bucket.backups.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "backups" {
  bucket = aws_s3_bucket.backups.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# A backup removed or replaced by mistake stays as a noncurrent version
# until the lifecycle deletes it.
resource "aws_s3_bucket_versioning" "backups" {
  bucket = aws_s3_bucket.backups.id
  versioning_configuration {
    status = "Enabled"
  }
}

# Expiration makes a backup noncurrent after retention_days, behind a delete
# marker; NoncurrentVersionExpiration deletes it for good one day later, and
# the delete markers left without versions are removed (Amazon S3 User
# Guide, intro-lifecycle-rules). expired_object_delete_marker conflicts with
# days, hence the second rule.
resource "aws_s3_bucket_lifecycle_configuration" "backups" {
  depends_on = [aws_s3_bucket_versioning.backups]
  bucket     = aws_s3_bucket.backups.id

  rule {
    id     = "expire-backups"
    status = "Enabled"
    filter {
      prefix = "keys/"
    }
    expiration {
      days = var.retention_days
    }
    noncurrent_version_expiration {
      noncurrent_days = 1
    }
  }

  # A dump resealed for a restore in production (#297) is deleted by the
  # owner once restored; this rule removes one left behind within two days.
  rule {
    id     = "expire-restores"
    status = "Enabled"
    filter {
      prefix = "restore/"
    }
    expiration {
      days = 1
    }
    noncurrent_version_expiration {
      noncurrent_days = 1
    }
  }

  rule {
    id     = "clean-up"
    status = "Enabled"
    filter {}
    expiration {
      expired_object_delete_marker = true
    }
    abort_incomplete_multipart_upload {
      days_after_initiation = 1
    }
  }
}

resource "aws_s3_bucket_policy" "backups" {
  bucket = aws_s3_bucket.backups.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "TLSOnly"
      Effect    = "Deny"
      Principal = "*"
      Action    = "s3:*"
      Resource  = [aws_s3_bucket.backups.arn, "${aws_s3_bucket.backups.arn}/*"]
      Condition = { Bool = { "aws:SecureTransport" = "false" } }
    }]
  })
}
