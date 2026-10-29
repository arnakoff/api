// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"errors"
	"net/mail"
	"strings"
)

// ErrInvalidEmail means the input is not a single email address with a
// valid domain. Its message never quotes the input, which is personal data.
var ErrInvalidEmail = errors.New("invalid email address")

// maxLocalLength is the longest local part RFC 5321 (4.5.3.1.1) allows, in
// octets.
const maxLocalLength = 64

// Email is an email address: the local part as written, the domain
// normalised by ParseDomain's rules. It is personal data: log Redacted, never
// String. The zero value is not a valid address.
type Email struct {
	local  string
	domain Domain
}

// ParseEmail reads a single RFC 5322 address, as a user would paste it from a
// message header ("Service client <contact@exemple.fr>"), and keeps the
// address only. The local part keeps its case, since RFC 5321 lets the
// receiving server tell cases apart. Nothing is checked over the network.
func ParseEmail(s string) (Email, error) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		// The cause quotes the input: drop it.
		return Email{}, ErrInvalidEmail
	}
	at := strings.LastIndexByte(addr.Address, '@')
	local, host := addr.Address[:at], addr.Address[at+1:]
	if local == "" || len(local) > maxLocalLength {
		return Email{}, ErrInvalidEmail
	}
	// A bare host name: the domain part may hold characters, such as '#',
	// that a URL parser would read as a delimiter.
	d, err := parseHostname(host)
	if err != nil {
		return Email{}, ErrInvalidEmail
	}
	return Email{local: local, domain: d}, nil
}

// String returns the address, such as "Jean.Dupont@exemple.fr", with the
// local part quoted when RFC 5322 requires it.
func (e Email) String() string {
	// With no display name, mail.Address.String returns "<addr-spec>",
	// quoting the local part as needed.
	quoted := (&mail.Address{Address: e.local + "@" + e.domain.String()}).String()
	return strings.TrimSuffix(strings.TrimPrefix(quoted, "<"), ">")
}

// Local returns the local part, unquoted.
func (e Email) Local() string {
	return e.local
}

// Domain returns the domain part.
func (e Email) Domain() Domain {
	return e.domain
}

// Redacted returns a form safe for logs, the domain only:
// "***@exemple.fr".
func (e Email) Redacted() string {
	return "***@" + e.domain.String()
}
