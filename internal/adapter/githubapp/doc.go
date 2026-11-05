// SPDX-License-Identifier: AGPL-3.0-only

// Package githubapp writes the secrets of a GitHub environment as a GitHub
// App, for the key rotation job (ADR 0007): it signs a JWT of the App with
// its private key, trades it for an installation token restricted to one
// repository and to writing its environments, and seals each secret for
// the public key of the environment.
//
// Every request goes to api.github.com through the platform HTTP client.
// No error of this package quotes the private key, a token or a secret.
package githubapp
