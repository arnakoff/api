// SPDX-License-Identifier: AGPL-3.0-only

package sirene

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// dateLayout is the layout of the API's dates, such as "1983-03-01".
const dateLayout = time.DateOnly

// Head office states (etat_administratif of an établissement).
const (
	stateOpen   = "A"
	stateClosed = "F"
)

// searchResponse holds the only fields read from a search answer. Results
// is a pointer so that an answer without it is told from an empty one.
type searchResponse struct {
	Results *[]result `json:"results"`
}

type result struct {
	SIREN      string     `json:"siren"`
	Name       string     `json:"nom_complet"`
	Created    *string    `json:"date_creation"`
	HeadOffice headOffice `json:"siege"`
}

type headOffice struct {
	State  string  `json:"etat_administratif"`
	Closed *string `json:"date_fermeture"`
}

// parseCompany reads the company with SIREN n from a search answer. The
// search is full text, so a result about another company is skipped. It
// returns ErrNotFound or ErrResponse.
func parseCompany(body []byte, n string) (domain.Company, error) {
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return domain.Company{}, fmt.Errorf("%w: %w", ErrResponse, err)
	}
	if resp.Results == nil {
		return domain.Company{}, fmt.Errorf("%w: no results field", ErrResponse)
	}
	for _, r := range *resp.Results {
		if r.SIREN == n {
			return r.company()
		}
	}
	return domain.Company{}, ErrNotFound
}

// company checks r and turns it into a domain.Company.
func (r result) company() (domain.Company, error) {
	if r.Name == "" {
		return domain.Company{}, fmt.Errorf("%w: no name", ErrResponse)
	}
	if r.HeadOffice.State != stateOpen && r.HeadOffice.State != stateClosed {
		return domain.Company{}, fmt.Errorf("%w: head office state %q", ErrResponse, r.HeadOffice.State)
	}
	created, err := parseDate(r.Created)
	if err != nil {
		return domain.Company{}, fmt.Errorf("%w: creation date: %w", ErrResponse, err)
	}
	closed, err := parseDate(r.HeadOffice.Closed)
	if err != nil {
		return domain.Company{}, fmt.Errorf("%w: closing date: %w", ErrResponse, err)
	}
	return domain.Company{
		Name:               r.Name,
		Created:            created,
		HeadOfficeClosed:   r.HeadOffice.State == stateClosed,
		HeadOfficeClosedOn: closed,
	}, nil
}

// parseDate reads an optional date; null gives the zero time.
func parseDate(s *string) (time.Time, error) {
	if s == nil {
		return time.Time{}, nil
	}
	return time.Parse(dateLayout, *s)
}
