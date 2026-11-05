// SPDX-License-Identifier: AGPL-3.0-only

// Package scaleway is the client of the Scaleway API the key rotation job
// uses (ADR 0007): it creates and deletes the API keys of a closed list of
// IAM applications, writes them to Secret Manager and to the secrets of a
// Serverless Container.
//
// Every request goes to api.scaleway.com through the platform HTTP client,
// with the key of the rotator. No error, log line or request description
// of this package quotes a secret key.
package scaleway
