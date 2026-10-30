// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

// Company holds what the SIRENE register publishes about a company and its
// head office (siège), and nothing about the people who run it. A zero time
// means the register did not publish that date.
type Company struct {
	// Name is the company's full name, as SIRENE writes it.
	Name string
	// Created is when the company was created.
	Created time.Time
	// HeadOfficeClosed reports whether SIRENE lists the head office as
	// closed.
	HeadOfficeClosed bool
	// HeadOfficeClosedOn is when the head office closed.
	HeadOfficeClosedOn time.Time
}
