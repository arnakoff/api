# SPDX-License-Identifier: AGPL-3.0-only
#
# The bucket of the backups of the API keys, out of Scaleway (#75): Amazon
# S3 in Paris (eu-west-3). Applied by the owner from their machine, never by
# the CI: see docs/deploy.md.

terraform {
  required_version = "~> 1.13.2"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.71.0"
    }
  }

  # State next to that of deploy/tofu, in the Scaleway bucket reserved to
  # the owner, under its own key. Its credentials are the owner's Scaleway
  # API key, from the profile scaleway-arnakoff of the shared AWS
  # credentials file, so that the AWS provider below never picks them up.
  backend "s3" {
    bucket                      = "arnakoff-tofu-state"
    key                         = "aws.tfstate"
    region                      = "fr-par"
    profile                     = "scaleway-arnakoff"
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    use_lockfile                = true
    endpoints = {
      s3 = "https://s3.fr-par.scw.cloud"
    }
  }

  # The state holds the access key of the backup container: encrypted on
  # the owner's machine before it is written, as in deploy/tofu.
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

# The account arnakoff-production of the organisation arnakoff, through the
# IAM Identity Center profile arnakoff-production (`aws sso login`): the
# provider reads SSO profiles from the shared configuration file (AWS SDK
# for Go v2, service/sso in its go.mod). allowed_account_ids stops a run
# against arnakoff-val or any other account before it changes anything.
provider "aws" {
  region              = "eu-west-3"
  profile             = "arnakoff-production"
  allowed_account_ids = [var.aws_account_id]

  default_tags {
    tags = {
      project = "arnakoff"
      source  = "github.com/arnakoff/api/deploy/tofu-aws"
    }
  }
}
