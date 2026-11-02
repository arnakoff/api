// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// ipv6ClientBits is the prefix length an IPv6 client is known by: a
// subscriber usually gets a whole /64, and would otherwise get as many
// quotas as it has addresses.
const ipv6ClientBits = 64

// clientKey returns the key the client of r is limited by: its address,
// or the /64 prefix of an IPv6 address. Every request whose address cannot
// be read shares one key.
func clientKey(r *http.Request, trusted []netip.Prefix) string {
	a := clientAddr(r, trusted)
	if a.Is6() {
		return netip.PrefixFrom(a, ipv6ClientBits).Masked().String()
	}
	return a.String()
}

// clientAddr returns the address of the client of r: the peer address,
// unless it is a trusted proxy. Then it walks X-Forwarded-For from the
// right, the hops each proxy appended, to the first address that is not a
// trusted proxy: the addresses left of it are whatever the client sent. A
// malformed hop stops the walk at the last proxy.
func clientAddr(r *http.Request, trusted []netip.Prefix) netip.Addr {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	addr := ap.Addr().Unmap()
	if !isTrusted(addr, trusted) {
		return addr
	}
	for _, hop := range slices.Backward(forwardedHops(r.Header)) {
		if !isTrusted(addr, trusted) {
			return addr
		}
		next, err := netip.ParseAddr(strings.TrimSpace(hop))
		if err != nil {
			return addr
		}
		addr = next.WithZone("").Unmap()
	}
	return addr
}

// forwardedHops returns the addresses of the X-Forwarded-For fields of h,
// in order, its several lines joined as one list (RFC 9110, section 5.3).
func forwardedHops(h http.Header) []string {
	var hops []string
	for _, line := range h.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(line, ",")...)
	}
	return hops
}

// isTrusted reports whether a is in one of the trusted prefixes.
func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	return slices.ContainsFunc(trusted, func(p netip.Prefix) bool { return p.Contains(a) })
}
