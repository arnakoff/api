// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import "errors"

var (
	// ErrThrottled means no request was sent: the API asked to wait
	// (Retry-After) and the wait is not over.
	ErrThrottled = errors.New("BODACC request rate exceeded")

	// ErrResponse means the API's answer is not a list of notices that can
	// be read.
	ErrResponse = errors.New("invalid BODACC records response")
)
