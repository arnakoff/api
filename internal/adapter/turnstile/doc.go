// SPDX-License-Identifier: AGPL-3.0-only

// Package turnstile checks a Cloudflare Turnstile token with the
// siteverify API, as documented at
// https://developers.cloudflare.com/turnstile/get-started/server-side-validation/:
// a form POST of the widget's secret key and the token, answered by a JSON
// object. The visitor's address, optional, is never sent.
//
// A token is valid once and for 300 seconds, so a check is never retried:
// a second try of the same token is refused as a duplicate. The secret and
// the token are never logged nor quoted in an error.
package turnstile
