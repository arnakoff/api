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
// "non_applicable", the values of the public contract.
func (s SourceStatus) String() string {
	switch s {
	case SourceOK:
		return "ok"
	case SourceUnavailable:
		return "indisponible"
	case SourceNotApplicable:
		return "non_applicable"
	}
	return "SourceStatus(" + strconv.Itoa(int(s)) + ")"
}

// SourceState is the status of one source in a report.
type SourceState struct {
	Source Source
	Status SourceStatus
}

// SourceInfo is what a source is, as GET /v1/sources shows it: who
// publishes it, under which licence it is reused and where to read it.
type SourceInfo struct {
	Name Source
	// Publisher is the body that publishes the source.
	Publisher string
	// Licence is the licence the source is reused under.
	Licence string
	// URL is the public page of the source.
	URL string
}
