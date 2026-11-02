// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/arnakoff/api/internal/domain"
)

// ErrInvalidOfficial means a row of the reference list of official domains
// is malformed.
var ErrInvalidOfficial = errors.New("invalid official domain")

// nameWords matches a name of the reference list: lower-case ASCII words
// joined by hyphens, as in a domain label.
var nameWords = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Official is a row of the reference list of official sites often
// imitated, each verified on a page of its operator.
type Official struct {
	// Domain is the registrable domain of the official site ("ameli.fr").
	Domain string
	// Alternates are other registrable domains of the same operator, which
	// are not look-alikes ("credit-agricole.com").
	Alternates []string
	// Names are spellings of the operator's name that a look-alike host
	// holds as words ("ameli", "creditagricole"). A label left out of them
	// ("cic", an ordinary acronym) matches its homoglyphs only.
	Names []string
	// Of names the operator in the message, after « site officiel » ("de
	// l'Assurance Maladie").
	Of string
	// Proof is the HTTPS page that shows the operator runs Domain, the
	// source_url of the signal.
	Proof string
}

// domains returns the registrable domains of o: its Domain, then its
// Alternates.
func (o Official) domains() []string {
	return append([]string{o.Domain}, o.Alternates...)
}

// reference is an Official prepared for matching: the skeletons of its
// labels and of the words of its names.
type reference struct {
	Official
	labels []label    // the labels of its domains
	names  [][]string // the skeleton words of each name
}

// label is the label of an official registrable domain, before its public
// suffix, prepared for matching.
type label struct {
	ascii    string
	skeleton []rune
	near     bool // whether a label one edit away matches
}

// Officials is the reference list of official domains, prepared for
// Lookalike. The zero value is an empty list.
type Officials struct {
	refs []reference
	own  map[string]struct{} // every registrable domain of the list
}

// NewOfficials returns the reference list of rows, in their order, or an
// error matching ErrInvalidOfficial for a malformed row or a domain listed
// twice.
func NewOfficials(rows []Official) (Officials, error) {
	o := Officials{refs: make([]reference, 0, len(rows)), own: make(map[string]struct{})}
	for _, row := range rows {
		ref, err := prepare(row)
		if err != nil {
			return Officials{}, fmt.Errorf("%w %q: %w", ErrInvalidOfficial, row.Domain, err)
		}
		for _, d := range row.domains() {
			if _, dup := o.own[d]; dup {
				return Officials{}, fmt.Errorf("%w %q: %s listed twice", ErrInvalidOfficial, row.Domain, d)
			}
			o.own[d] = struct{}{}
		}
		o.refs = append(o.refs, ref)
	}
	return o, nil
}

// prepare checks row and computes its skeletons.
func prepare(row Official) (reference, error) {
	if row.Of == "" {
		return reference{}, errors.New("no operator name")
	}
	if u, err := url.Parse(row.Proof); err != nil || u.Scheme != "https" || u.Host == "" {
		return reference{}, fmt.Errorf("proof %q is not an HTTPS URL", row.Proof)
	}
	ref := reference{Official: row}
	for _, d := range row.domains() {
		l, err := officialLabel(d)
		if err != nil {
			return reference{}, err
		}
		ref.labels = append(ref.labels, l)
	}
	for _, n := range row.Names {
		if !nameWords.MatchString(n) {
			return reference{}, fmt.Errorf("name %q is not lower-case words joined by hyphens", n)
		}
		ref.names = append(ref.names, skeletonWords(n, isWordSeparator))
	}
	return ref, nil
}

// officialLabel returns the label of d, which must be a normalised
// registrable domain.
func officialLabel(d string) (label, error) {
	parsed, err := domain.ParseDomain(d)
	if err != nil {
		return label{}, err
	}
	if parsed.String() != d || parsed.Registrable() != d {
		return label{}, fmt.Errorf("%q is not a normalised registrable domain", d)
	}
	ascii := parsed.Label()
	return label{ascii: ascii, skeleton: []rune(skeleton(ascii)), near: len(ascii) >= NearLabelLength}, nil
}
