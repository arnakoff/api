// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

// Domain errors. ErrUnsupportedScheme and ErrIPAddress also match
// ErrInvalidDomain with errors.Is.
var (
	// ErrInvalidDomain means the input is not a domain name or a web URL.
	ErrInvalidDomain = errors.New("invalid domain")
	// ErrUnsupportedScheme means the input is a URL whose scheme is neither
	// http nor https.
	ErrUnsupportedScheme = fmt.Errorf("%w: unsupported URL scheme", ErrInvalidDomain)
	// ErrIPAddress means the host is an IP address, not a domain name.
	ErrIPAddress = fmt.Errorf("%w: IP address", ErrInvalidDomain)
)

// idnaProfile converts names to their ASCII form. It is spelled out rather
// than taken from idna.Lookup, whose configuration "may change over time":
// the UTS #46 lookup mapping (lower case, width, NFC), STD3 characters,
// hyphen and joiner rules, the Bidi rule, non-transitional processing and
// the DNS lengths (253 characters, 63 per label). It is immutable.
var idnaProfile = idna.New(
	idna.MapForLookup(),
	idna.BidiRule(),
	idna.Transitional(false),
	idna.VerifyDNSLength(true),
)

// Domain is a normalised domain name: lower case, A-labels, no final dot.
// The zero value is not a valid domain.
type Domain struct {
	name        string
	registrable string
}

// ParseDomain normalises a domain name or a web URL, as a user would paste
// it, into a Domain. A URL keeps its host only: credentials before the host
// are dropped, as a browser does ("https://amf-france.org@exemple.com" is
// exemple.com). Only http and https URLs are accepted, and IP addresses are
// reported with ErrIPAddress.
func ParseDomain(s string) (Domain, error) {
	host, err := hostOf(strings.TrimSpace(s))
	if err != nil {
		return Domain{}, err
	}
	name, err := normaliseName(host)
	if err != nil {
		return Domain{}, err
	}
	registrable, err := registrableOf(name)
	if err != nil {
		return Domain{}, err
	}
	return Domain{name: name, registrable: registrable}, nil
}

// String returns the normalised name, such as "www.groupe-ytria.com".
func (d Domain) String() string {
	return d.name
}

// Registrable returns the public suffix plus one label, the part a
// registrant controls: "groupe-ytria.com" for www.groupe-ytria.com,
// "atos-limited.co.uk" for www.atos-limited.co.uk. Private suffixes of the
// Public Suffix List count, so arnaque.github.io is its own registrable
// domain.
func (d Domain) Registrable() string {
	return d.registrable
}

// hostOf returns the host of s, read as a URL when it starts with a scheme
// and as a URL without scheme otherwise.
func hostOf(s string) (string, error) {
	// Browsers read a backslash as a slash in http and https URLs.
	s = strings.ReplaceAll(s, `\`, "/")
	if !hasScheme(s) {
		s = "//" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		// Keep the cause, not the *url.Error that quotes the whole input.
		if urlErr := (*url.Error)(nil); errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return "", fmt.Errorf("%w: %w", ErrInvalidDomain, err)
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%w %q", ErrUnsupportedScheme, u.Scheme)
	}
	host := u.Hostname()
	if _, err := netip.ParseAddr(host); err == nil {
		return "", ErrIPAddress
	}
	return host, nil
}

// hasScheme reports whether s starts with "<scheme>://" (RFC 3986, 3.1).
func hasScheme(s string) bool {
	scheme, _, found := strings.Cut(s, "://")
	if !found || scheme == "" || !isASCIILetter(scheme[0]) {
		return false
	}
	for i := 1; i < len(scheme); i++ {
		c := scheme[i]
		if !isASCIILetter(c) && !isASCIIDigit(c) && c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

// normaliseName converts host to its ASCII form and checks its lengths and
// characters.
func normaliseName(host string) (string, error) {
	host = strings.TrimSuffix(host, ".")
	name, err := idnaProfile.ToASCII(host)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidDomain, err)
	}
	if endsInNumber(name) {
		return "", ErrIPAddress
	}
	return name, nil
}

// endsInNumber reports whether the last label of name is a number, which
// browsers read as an IPv4 address ("http://3221225994/"), as in the WHATWG
// URL standard.
func endsInNumber(name string) bool {
	label := name[strings.LastIndexByte(name, '.')+1:]
	if hex, ok := strings.CutPrefix(label, "0x"); ok {
		return strings.Trim(hex, "0123456789abcdef") == ""
	}
	return strings.Trim(label, "0123456789") == ""
}

// registrableOf returns the public suffix of name plus one label. It rejects
// a name under a TLD missing from the Public Suffix List and a name that is
// itself a public suffix.
func registrableOf(name string) (string, error) {
	suffix, icann := publicsuffix.PublicSuffix(name)
	if !icann && !strings.Contains(suffix, ".") {
		return "", fmt.Errorf("%w: unknown top-level domain %q", ErrInvalidDomain, suffix)
	}
	registrable, err := publicsuffix.EffectiveTLDPlusOne(name)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidDomain, err)
	}
	return registrable, nil
}

func isASCIILetter(c byte) bool { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' }

func isASCIIDigit(c byte) bool { return '0' <= c && c <= '9' }
