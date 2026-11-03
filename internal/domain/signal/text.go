// SPDX-License-Identifier: AGPL-3.0-only

package signal

import (
	"unicode"
	"unicode/utf8"
)

// MaxMessageRunes bounds the message of a signal, quoted third-party text
// included: a longer one is cut and ends with "…".
const MaxMessageRunes = 1000

// visible returns message as its reader sees it. A message quotes text
// that third parties chose (a company name, an operator, a category), and
// assistants read it: a character that shows nothing or reorders the text
// (format, private use, variation selector) is dropped, so no hidden text
// reaches them, and a control or line separator becomes a space, so the
// text makes no line of its own. Invalid UTF-8 becomes U+FFFD. The result
// is cut to MaxMessageRunes. A message already visible is returned as is,
// without allocating: it is the common case, on the hot path.
func visible(message string) string {
	if isVisible(message) {
		return message
	}
	runes := make([]rune, 0, len(message))
	for _, r := range message {
		switch {
		case hidden(r):
			continue
		case breaking(r):
			r = ' '
		}
		runes = append(runes, r)
	}
	if len(runes) > MaxMessageRunes {
		runes = append(runes[:MaxMessageRunes-1], '…')
	}
	return string(runes)
}

// isVisible reports whether visible would leave message unchanged. ASCII
// is checked first: only its controls change.
func isVisible(message string) bool {
	n := 0
	for i, r := range message {
		n++
		switch {
		case r < utf8.RuneSelf:
			if r < ' ' || r == 0x7f {
				return false
			}
		case latin(r):
		case r == utf8.RuneError:
			if _, size := utf8.DecodeRuneInString(message[i:]); size == 1 {
				return false
			}
		case hidden(r) || breaking(r):
			return false
		}
	}
	return n <= MaxMessageRunes
}

// latin reports whether r is one of the Latin-1 Supplement and Latin
// Extended-A and -B characters (U+00A0 to U+024F) French text uses, all
// visible but the soft hyphen U+00AD: a fast answer, before the Unicode
// tables.
func latin(r rune) bool {
	return r >= 0xA0 && r <= 0x24F && r != 0xAD
}

// hidden reports whether r shows nothing or reorders the text around it.
func hidden(r rune) bool {
	return unicode.In(r, unicode.Cf, unicode.Co, unicode.Variation_Selector)
}

// breaking reports whether r controls the terminal or breaks the line.
func breaking(r rune) bool {
	return unicode.In(r, unicode.Cc, unicode.Zl, unicode.Zp)
}
