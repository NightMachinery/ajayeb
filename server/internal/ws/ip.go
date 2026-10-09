package ws

import (
	"net/http"
	"net/netip"
	"strings"
)

// clientIP returns the address a request comes from, for the per-IP limits.
//
// It is the TCP peer, unless the peer is a trusted proxy. Then the
// X-Forwarded-For header is read from the right: every trusted proxy
// appends the address it received the request from, so the rightmost entry
// that is not itself a trusted proxy is the client. Entries to the left of
// that one were written by the client and are never believed. A malformed
// entry stops the walk at the last address that was believable. A peer that
// is not trusted cannot set the address at all, whatever it sends.
func clientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	ip := parseHostAddr(r.RemoteAddr)
	if !ip.IsValid() || !isTrusted(ip, trusted) {
		return ip
	}
	hops := forwardedHops(r.Header.Values("X-Forwarded-For"))
	for i := len(hops) - 1; i >= 0; i-- {
		a := parseHostAddr(hops[i])
		if !a.IsValid() {
			break
		}
		ip = a
		if !isTrusted(a, trusted) {
			break
		}
	}
	return ip
}

// forwardedHops splits X-Forwarded-For header values into their entries,
// in order. A request may carry the header more than once; the values then
// read as one comma separated list.
func forwardedHops(values []string) []string {
	var hops []string
	for _, v := range values {
		for hop := range strings.SplitSeq(v, ",") {
			hops = append(hops, strings.TrimSpace(hop))
		}
	}
	return hops
}

// parseHostAddr parses an address with or without a port ("192.0.2.1",
// "192.0.2.1:5000", "::1", "[::1]:5000"), unmapping IPv4-in-IPv6. It
// returns the zero Addr when s is neither.
func parseHostAddr(s string) netip.Addr {
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap()
	}
	a, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(s, "["), "]"))
	if err != nil {
		return netip.Addr{}
	}
	return a.Unmap()
}

// isTrusted reports whether a is inside one of the trusted prefixes.
func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// originPatterns turns the configured site origins into the patterns
// websocket.Accept checks the Origin header against. The request's own host
// is always accepted by Accept itself, and a request without an Origin
// header (a tool, not a browser) is never cross-site.
//
// A configured origin is matched exactly, scheme included: its
// path.Match metacharacters are escaped, so an IPv6 origin such as
// http://[::1]:8080 is not read as a character class. In development the
// loopback names are added on any port (and without one), so the Vite dev
// server, whose pages come from another port, can connect.
func originPatterns(origins []string, dev bool) []string {
	out := make([]string, 0, len(origins)+4)
	for _, o := range origins {
		out = append(out, escapePattern(o))
	}
	if dev {
		out = append(out, "localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*")
	}
	return out
}

// escapePattern escapes the metacharacters of path.Match.
func escapePattern(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '*', '?', '[', ']', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
