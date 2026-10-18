package middleware

import (
	"net/http"
	"net/netip"
	"strings"
)

func normalize(a netip.Addr) netip.Addr {
	return a.Unmap().WithZone("")
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

func parseAddr(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return netip.Addr{}, false
	}
	return normalize(a), true
}

// ClientIP resolves the client address of r. Forwarding headers are honoured
// only when the direct peer is inside one of the trusted prefixes; the
// X-Forwarded-For chain is then walked right to left and the first address not
// belonging to a trusted proxy is returned. The result is the canonical string
// form of the address. If RemoteAddr cannot be parsed it is returned unchanged.
func ClientIP(r *http.Request, trusted []netip.Prefix) string {
	var remote netip.Addr
	if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		remote = normalize(ap.Addr())
	} else if a, err := netip.ParseAddr(r.RemoteAddr); err == nil {
		remote = normalize(a)
	} else {
		return r.RemoteAddr
	}
	if !isTrusted(remote, trusted) {
		return remote.String()
	}

	if lines := r.Header.Values("X-Forwarded-For"); len(lines) > 0 {
		entries := strings.Split(strings.Join(lines, ","), ",")
		prev := remote
		for i := len(entries) - 1; i >= 0; i-- {
			a, ok := parseAddr(entries[i])
			if !ok {
				return prev.String()
			}
			if !isTrusted(a, trusted) {
				return a.String()
			}
			prev = a
		}
		return prev.String()
	}

	if a, ok := parseAddr(r.Header.Get("X-Real-IP")); ok {
		return a.String()
	}
	return remote.String()
}
