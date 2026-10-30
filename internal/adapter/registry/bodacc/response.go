// SPDX-License-Identifier: AGPL-3.0-only

package bodacc

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/arnakoff/api/internal/domain"
)

// dateLayout is the layout of the API's dates, such as "2026-10-30".
const dateLayout = time.DateOnly

// Notice types (typeavis): a correction or a cancellation points to the
// notice it corrects or cancels.
const (
	typeNotice       = "annonce"
	typeCorrection   = "rectificatif"
	typeCancellation = "annulation"
)

// kinds maps the families asked for (familleavis) to their kind.
var kinds = map[string]domain.NoticeKind{
	"collective":                   domain.NoticeCollectiveProcedure,
	"radiation":                    domain.NoticeRadiation,
	"conciliation":                 domain.NoticeConciliation,
	"retablissement_professionnel": domain.NoticeProfessionalRecovery,
}

// recordsResponse holds the only fields read from a records answer.
// Results is a pointer so that an answer without it is told from an empty
// one.
type recordsResponse struct {
	Results *[]entry `json:"results"`
}

// entry is a notice as the API returns it, with the fields selected.
type entry struct {
	ID          string   `json:"id"`
	Registers   []string `json:"registre"`
	Published   string   `json:"dateparution"`
	Family      string   `json:"familleavis"`
	Type        string   `json:"typeavis"`
	Publication string   `json:"publicationavis"`
	Parution    string   `json:"parution"`
	Number      int      `json:"numeroannonce"`
	Precedent   *string  `json:"parutionavisprecedent"`
}

// precedentRef is the parutionavisprecedent of a correction or a
// cancellation, a JSON object held in a string.
type precedentRef struct {
	Publication string `json:"nomPublication"`
	Parution    string `json:"numeroParution"`
	Number      string `json:"numeroAnnonce"`
}

// ref identifies a notice: its BODACC (A, B or C), its parution and its
// number in that parution.
type ref struct {
	publication string
	parution    string
	number      int
}

// parseNotices reads the notices about the company with SIREN n from a
// records answer, in the answer's order. A notice about other companies
// only is skipped; a cancelled or corrected notice is left out, and so is
// the cancellation. It returns ErrResponse.
func parseNotices(body []byte, n string) ([]domain.Notice, error) {
	var resp recordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResponse, err)
	}
	if resp.Results == nil {
		return nil, fmt.Errorf("%w: no results field", ErrResponse)
	}
	entries := slices.DeleteFunc(*resp.Results, func(e entry) bool { return !slices.Contains(e.Registers, n) })
	superseded, err := supersededRefs(entries)
	if err != nil {
		return nil, err
	}
	notices := make([]domain.Notice, 0, len(entries))
	for _, e := range entries {
		if e.Type == typeCancellation || superseded[e.ref()] {
			continue
		}
		notice, err := e.notice()
		if err != nil {
			return nil, err
		}
		notices = append(notices, notice)
	}
	return notices, nil
}

// supersededRefs returns the notices that entries correct or cancel.
func supersededRefs(entries []entry) (map[ref]bool, error) {
	superseded := make(map[ref]bool)
	for _, e := range entries {
		r, ok, err := e.supersedes()
		if err != nil {
			return nil, fmt.Errorf("%w: notice %s: %w", ErrResponse, e.ID, err)
		}
		if ok {
			superseded[r] = true
		}
	}
	return superseded, nil
}

func (e entry) ref() ref {
	return ref{publication: e.Publication, parution: e.Parution, number: e.Number}
}

// supersedes returns the notice e corrects or cancels, if it is a
// correction or a cancellation.
func (e entry) supersedes() (ref, bool, error) {
	switch e.Type {
	case typeNotice:
		return ref{}, false, nil
	case typeCorrection, typeCancellation:
		if e.Precedent == nil {
			return ref{}, false, fmt.Errorf("%s without precedent", e.Type)
		}
		r, err := parsePrecedent(*e.Precedent)
		return r, err == nil, err
	default:
		return ref{}, false, fmt.Errorf("notice type %q", e.Type)
	}
}

// parsePrecedent reads a parutionavisprecedent, such as {"nomPublication":
// "BODACC A", "numeroParution": "20250032", "numeroAnnonce": "2546"}.
func parsePrecedent(s string) (ref, error) {
	var p precedentRef
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return ref{}, fmt.Errorf("precedent: %w", err)
	}
	publication, ok := strings.CutPrefix(p.Publication, "BODACC ")
	if !ok || !slices.Contains([]string{"A", "B", "C"}, publication) {
		return ref{}, fmt.Errorf("precedent publication %q", p.Publication)
	}
	number, err := strconv.Atoi(p.Number)
	if err != nil {
		return ref{}, fmt.Errorf("precedent number: %w", err)
	}
	return ref{publication: publication, parution: p.Parution, number: number}, nil
}

// notice checks e and turns it into a domain.Notice.
func (e entry) notice() (domain.Notice, error) {
	if e.ID == "" {
		return domain.Notice{}, fmt.Errorf("%w: notice without id", ErrResponse)
	}
	kind, ok := kinds[e.Family]
	if !ok {
		return domain.Notice{}, fmt.Errorf("%w: notice %s: family %q", ErrResponse, e.ID, e.Family)
	}
	published, err := time.Parse(dateLayout, e.Published)
	if err != nil {
		return domain.Notice{}, fmt.Errorf("%w: notice %s: publication date: %w", ErrResponse, e.ID, err)
	}
	return domain.Notice{ID: e.ID, Kind: kind, Published: published}, nil
}
