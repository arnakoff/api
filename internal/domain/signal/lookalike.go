// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"slices"
	"strings"
	"time"

	"go.mau.fi/util/confusable"

	"github.com/arnakoff/api/internal/domain"
)

// LookalikeSource names the source of domain_lookalike: the reference list
// of official domains.
const LookalikeSource domain.Source = "Domaines officiels"

// NearLabelLength is the length from which an official label also matches
// a label one edit away: shorter labels are too close to ordinary words
// ("ameli" to "amelie").
const NearLabelLength = 6

// lookalikeAdvice is the advice of domain_lookalike.
const lookalikeAdvice = "Tape toi-même l'adresse officielle plutôt que de suivre un lien reçu."

// Lookalike returns the attention stating that d resembles the first
// official domain of the list it imitates, observed at now; false when d
// belongs to an official domain or resembles none. d imitates an official
// domain when, compared by their UTS #39 skeletons in lower case:
//   - its label (Domain.Label) is an official label spelled otherwise, a
//     homoglyph;
//   - or it is one edit (insertion, deletion, substitution, adjacent swap)
//     away from an official label of NearLabelLength letters or more;
//   - or its host, less the public suffix, holds a listed name as a run of
//     whole words, words being separated by dots and hyphens. The names
//     decide whether the official label matches under another suffix.
func (o Officials) Lookalike(d domain.Domain, now time.Time) (domain.Signal, bool) {
	if len(o.refs) == 0 {
		return domain.Signal{}, false
	}
	if _, own := o.own[d.Registrable()]; own {
		return domain.Signal{}, false
	}
	words, label := hostWords(d)
	l := []rune(label)
	for _, ref := range o.refs {
		if ref.resembles(d.Label(), l) || ref.named(words) {
			return lookalikeSignal(d, ref.Official, now), true
		}
	}
	return domain.Signal{}, false
}

// resembles reports whether the label ascii, of skeleton l, is a homoglyph
// of one of ref's labels, or near one.
func (ref reference) resembles(ascii string, l []rune) bool {
	for _, own := range ref.labels {
		if ascii != own.ascii && slices.Equal(l, own.skeleton) {
			return true
		}
		if own.near && oneEditApart(l, own.skeleton) {
			return true
		}
	}
	return false
}

// named reports whether the skeleton words of a host hold one of ref's
// names as a run.
func (ref reference) named(words []string) bool {
	for _, name := range ref.names {
		for i := 0; i+len(name) <= len(words); i++ {
			if slices.Equal(words[i:i+len(name)], name) {
				return true
			}
		}
	}
	return false
}

// lookalikeSignal returns domain_lookalike for d imitating official.
func lookalikeSignal(d domain.Domain, official Official, now time.Time) domain.Signal {
	message := "Le domaine " + d.String() + " ressemble à " + official.Domain + ", site officiel " + official.Of + ", mais n'en fait pas partie."
	return domainLookalike.signal(message, lookalikeAdvice, official.Proof, now)
}

// hostWords returns the skeletons of the words of d's host less its public
// suffix, split at dots and hyphens, and the skeleton of its label. A host
// is decoded only when it holds an A-label.
func hostWords(d domain.Domain) (words []string, label string) {
	host := strings.TrimSuffix(d.String(), d.Registrable()) + d.Label()
	if strings.Contains(host, "xn--") {
		host = domain.DecodeName(host)
	}
	label = skeleton(host[strings.LastIndexByte(host, '.')+1:])
	return skeletonWords(host, isWordSeparator), label
}

// isWordSeparator reports whether c separates the words of a host: a dot
// or a hyphen.
func isWordSeparator(c rune) bool { return c == '.' || c == '-' }

// skeletonWords returns the skeletons of the words of s, split where
// separator reports true.
func skeletonWords(s string, separator func(rune) bool) []string {
	words := strings.FieldsFunc(s, separator)
	for i, w := range words {
		words[i] = skeleton(w)
	}
	return words
}

// skeleton returns the UTS #39 skeleton of s in lower case: domain names
// ignore case, so "0" (skeleton "O") matches "o".
func skeleton(s string) string {
	return strings.ToLower(confusable.Skeleton(s))
}

// oneEditApart reports whether x and y differ by exactly one insertion,
// deletion, substitution or swap of adjacent characters.
func oneEditApart(x, y []rune) bool {
	if len(x) > len(y) {
		x, y = y, x
	}
	switch len(y) - len(x) {
	case 0:
		return oneSubstitutionOrSwap(x, y)
	case 1:
		return oneInsertion(x, y)
	default:
		return false
	}
}

// oneSubstitutionOrSwap reports whether x and y, of the same length,
// differ by one substitution or one swap of adjacent characters.
func oneSubstitutionOrSwap(x, y []rune) bool {
	i := 0
	for i < len(x) && x[i] == y[i] {
		i++
	}
	if i == len(x) {
		return false
	}
	if slices.Equal(x[i+1:], y[i+1:]) {
		return true
	}
	return i+1 < len(x) && x[i] == y[i+1] && x[i+1] == y[i] && slices.Equal(x[i+2:], y[i+2:])
}

// oneInsertion reports whether y is x with one character inserted.
func oneInsertion(x, y []rune) bool {
	i := 0
	for i < len(x) && x[i] == y[i] {
		i++
	}
	return slices.Equal(x[i:], y[i+1:])
}
