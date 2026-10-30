// SPDX-License-Identifier: AGPL-3.0-only

package replay

import (
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// registration is the recorded form of a domain.Registration.
type registration struct {
	Registered  time.Time `json:"registered,omitzero"`
	LastChanged time.Time `json:"last_changed,omitzero"`
}

// company is the recorded form of a domain.Company.
type company struct {
	Name               string    `json:"name"`
	Created            time.Time `json:"created,omitzero"`
	HeadOfficeClosed   bool      `json:"head_office_closed,omitempty"`
	HeadOfficeClosedOn time.Time `json:"head_office_closed_on,omitzero"`
}

// notice is the recorded form of a domain.Notice.
type notice struct {
	ID        string            `json:"id"`
	Kind      domain.NoticeKind `json:"kind"`
	Published time.Time         `json:"published"`
}

func fromNotices(ns []domain.Notice) []notice {
	out := make([]notice, len(ns))
	for i, n := range ns {
		out[i] = notice(n)
	}
	return out
}

func toNotices(ns []notice) []domain.Notice {
	out := make([]domain.Notice, len(ns))
	for i, n := range ns {
		out[i] = domain.Notice(n)
	}
	return out
}
