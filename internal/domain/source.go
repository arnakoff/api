// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "strconv"

// Source names an upstream data source, as shown to users ("AMF", "SIRENE").
type Source string

// SourceStatus tells how a source took part in a check. The zero value is
// not a valid status.
type SourceStatus int

// Source statuses.
const (
	// SourceOK means the source was consulted.
	SourceOK SourceStatus = iota + 1
	// SourceUnavailable means the source could not be consulted: the absence
	// of a signal from it proves nothing.
	SourceUnavailable
	// SourceNotApplicable means the source does not cover this input type.
	SourceNotApplicable
)

// String returns the status's stable French code: "ok", "indisponible" or
// "non applicable".
func (s SourceStatus) String() string {
	switch s {
	case SourceOK:
		return "ok"
	case SourceUnavailable:
		return "indisponible"
	case SourceNotApplicable:
		return "non applicable"
	}
	return "SourceStatus(" + strconv.Itoa(int(s)) + ")"
}

// SourceState is the status of one source in a report.
type SourceState struct {
	Source Source
	Status SourceStatus
}
