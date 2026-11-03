# SPDX-License-Identifier: AGPL-3.0-only
#
# Production infrastructure on Scaleway, fr-par (ADR 0005, #71). Applied by
# the owner from their machine, never by the CI: see docs/deploy.md.

terraform {
  required_version = "~> 1.13.2"

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "2.89.0"
    }
  }

  # State in a private, versioned bucket of the project, created once by
  # hand (docs/deploy.md). Block from Scaleway's Terraform/OpenTofu
  # quickstart (scaleway/docs-content, tutorials/terraform-quickstart); the
  # credentials come from AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.
  # use_lockfile locks with an S3 conditional write (If-None-Match), which
  # Scaleway Object Storage supports (pages/object-storage/api-cli/
  # using-conditional-writes.mdx, validated 03/07/2026).
  backend "s3" {
    bucket                      = "arnakoff-tofu-state"
    key                         = "production.tfstate"
    region                      = "fr-par"
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    use_lockfile                = true
    endpoints = {
      s3 = "https://s3.fr-par.scw.cloud"
    }
  }

  # The state holds the bucket keys: it is encrypted on the owner's machine
  # before it is written, with a key derived from var.state_passphrase
  # (OpenTofu state encryption, pbkdf2 and aes_gcm). Encryption is enforced:
  # a run without the passphrase fails instead of writing in clear.
  encryption {
    key_provider "pbkdf2" "owner" {
      passphrase = var.state_passphrase
    }
    method "aes_gcm" "owner" {
      keys = key_provider.pbkdf2.owner
    }
    state {
      method   = method.aes_gcm.owner
      enforced = true
    }
    plan {
      method   = method.aes_gcm.owner
      enforced = true
    }
  }
}

provider "scaleway" {
  region = "fr-par"
  zone   = "fr-par-1"
}
