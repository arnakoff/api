// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

// Authorisation is one entry of a white list: a provider or an offer the
// publisher authorised or registered, under a site, an address, a phone
// number or a SIREN.
type Authorisation struct {
	// Name is the provider or offer name, as the publisher lists it.
	Name string
	// Number is the publisher's number of the authorisation or offer.
	Number string
	// Kind is the kind of authorisation ("Agrément MICA"), "" when the list
	// has one kind only.
	Kind string
	// From is the day the authorisation started.
	From time.Time
	// Until is the day it ended, zero while it runs.
	Until time.Time
}

// Ended reports whether the authorisation has ended.
func (a Authorisation) Ended() bool { return !a.Until.IsZero() }

// Whitelist indexes the authorisations of one white list by the site,
// address, phone number or SIREN they name. Once built it is read-only:
// readers share it.
type Whitelist struct {
	dir directory[Authorisation]
}

// NewWhitelist returns an empty Whitelist.
func NewWhitelist() *Whitelist {
	return &Whitelist{}
}

// AddSite lists the whole site d: its host and every subdomain of it. A
// "www." host names its site, unless its site is a public suffix.
func (w *Whitelist) AddSite(d Domain, a Authorisation) { w.dir.addSite(d, a) }

// AddEmail lists the address e alone.
func (w *Whitelist) AddEmail(e Email, a Authorisation) { w.dir.addEmail(e, a) }

// AddPhone lists the phone number p.
func (w *Whitelist) AddPhone(p Phone, a Authorisation) { w.dir.addPhone(p, a) }

// AddSIREN lists the company n: its SIREN and every SIRET of it.
func (w *Whitelist) AddSIREN(n SIREN, a Authorisation) { w.dir.addSIREN(n, a) }

// Authorisations returns the authorisations that cover in, nil when none
// does. A listed site covers its subdomains, never its parent; a listed
// SIREN covers the SIRETs of the company. The caller must not change the
// authorisations it gets.
func (w *Whitelist) Authorisations(in Input) []Authorisation { return w.dir.lookup(in) }
