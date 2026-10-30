// SPDX-License-Identifier: AGPL-3.0-only

package rdap

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// Event actions read from a domain object (RFC 9083 section 4.5, values
// from the IANA RDAP JSON values registry).
const (
	actionRegistration = "registration"
	actionLastChanged  = "last changed"
)

// domainObject holds the only fields read from an RDAP domain object: its
// class and its own events. Entities, and so contacts, are never decoded.
type domainObject struct {
	ObjectClassName string `json:"objectClassName"`
	Events          []struct {
		Action string `json:"eventAction"`
		Date   string `json:"eventDate"`
	} `json:"events"`
}

// parseRegistration reads the registration and last changed dates of an
// RDAP domain object, or returns ErrResponse.
func parseRegistration(body []byte) (domain.Registration, error) {
	var obj domainObject
	if err := json.Unmarshal(body, &obj); err != nil {
		return domain.Registration{}, fmt.Errorf("%w: %w", ErrResponse, err)
	}
	if obj.ObjectClassName != "domain" {
		return domain.Registration{}, fmt.Errorf("%w: object class %q", ErrResponse, obj.ObjectClassName)
	}
	var reg domain.Registration
	for _, e := range obj.Events {
		var field *time.Time
		switch e.Action {
		case actionRegistration:
			field = &reg.Registered
		case actionLastChanged:
			field = &reg.LastChanged
		default:
			continue
		}
		t, err := parseEventDate(e.Date)
		if err != nil {
			return domain.Registration{}, fmt.Errorf("%w: %s date: %w", ErrResponse, e.Action, err)
		}
		*field = t
	}
	return reg, nil
}

// parseEventDate reads an RFC 3339 date-time, as RFC 9083 section 4.5
// requires, or a date alone, read as midnight UTC: some registries
// (rdap.register.si for .si) publish only the date.
func parseEventDate(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	if d, dErr := time.Parse(time.DateOnly, s); dErr == nil {
		return d, nil
	}
	return time.Time{}, err
}
