package ws

import (
	"net/http/httptest"
	"net/netip"
	"path"
	"slices"
	"strings"
	"testing"
)

func TestClientIP(t *testing.T) {
	proxy := []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32"), netip.MustParsePrefix("10.0.0.0/8")}
	for _, tc := range []struct {
		name    string
		remote  string
		xff     []string
		trusted []netip.Prefix
		want    string
	}{
		{"plain peer", "192.0.2.1:5000", nil, nil, "192.0.2.1"},
		{"untrusted peer ignores xff", "192.0.2.1:5000", []string{"198.51.100.7"}, proxy, "192.0.2.1"},
		{"no trusted proxies ignores xff", "127.0.0.1:5000", []string{"198.51.100.7"}, nil, "127.0.0.1"},
		{"trusted proxy", "127.0.0.1:5000", []string{"198.51.100.7"}, proxy, "198.51.100.7"},
		{"rightmost untrusted wins", "127.0.0.1:5000", []string{"203.0.113.9, 198.51.100.7"}, proxy, "198.51.100.7"},
		{"trusted hops are skipped", "127.0.0.1:5000", []string{"203.0.113.9, 198.51.100.7, 10.1.2.3"}, proxy, "198.51.100.7"},
		{"several headers read as one list", "127.0.0.1:5000", []string{"203.0.113.9", "198.51.100.7"}, proxy, "198.51.100.7"},
		{"all trusted gives the leftmost", "127.0.0.1:5000", []string{"10.0.0.5, 10.0.0.6"}, proxy, "10.0.0.5"},
		{"malformed entry stops the walk", "127.0.0.1:5000", []string{"203.0.113.9, junk, 10.0.0.6"}, proxy, "10.0.0.6"},
		{"malformed last entry keeps the peer", "127.0.0.1:5000", []string{"203.0.113.9, junk"}, proxy, "127.0.0.1"},
		{"empty header keeps the peer", "127.0.0.1:5000", []string{""}, proxy, "127.0.0.1"},
		{"entry with a port", "127.0.0.1:5000", []string{"198.51.100.7:4444"}, proxy, "198.51.100.7"},
		{"ipv6 peer", "[2001:db8::1]:5000", nil, proxy, "2001:db8::1"},
		{"ipv6 entry", "127.0.0.1:5000", []string{"2001:db8::7"}, proxy, "2001:db8::7"},
		{"mapped ipv4 peer is unmapped", "[::ffff:127.0.0.1]:5000", []string{"198.51.100.7"}, proxy, "198.51.100.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/ws", nil)
			r.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := clientIP(r, tc.trusted); got != netip.MustParseAddr(tc.want) {
				t.Errorf("clientIP = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClientIPUnparsablePeer(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.RemoteAddr = "pipe"
	r.Header.Set("X-Forwarded-For", "198.51.100.7")
	if got := clientIP(r, []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}); got.IsValid() {
		t.Errorf("clientIP = %v, want the zero address for an unparsable peer", got)
	}
}

func TestOriginPatterns(t *testing.T) {
	prod := originPatterns([]string{"https://ajayeb.example"}, false)
	if !slices.Equal(prod, []string{"https://ajayeb.example"}) {
		t.Errorf("production patterns = %q, want only the configured site", prod)
	}
	dev := originPatterns([]string{"https://ajayeb.example"}, true)
	for _, want := range []string{"localhost:*", "127.0.0.1:*", "localhost", "127.0.0.1"} {
		if !slices.Contains(dev, want) {
			t.Errorf("dev patterns %q lack %q", dev, want)
		}
	}
	// The patterns as websocket.Accept uses them: path.Match, lower case,
	// against scheme://host for a pattern with a scheme and host otherwise.
	matches := func(patterns []string, target string) bool {
		for _, p := range patterns {
			if ok, err := path.Match(strings.ToLower(p), strings.ToLower(target)); err != nil {
				t.Fatalf("pattern %q: %v", p, err)
			} else if ok {
				return true
			}
		}
		return false
	}
	if !matches(dev, "localhost:38952") || !matches(dev, "127.0.0.1:5173") || matches(dev, "localhost.evil.example:80") {
		t.Error("dev loopback patterns match the wrong hosts")
	}
	v6 := originPatterns([]string{"http://[::1]:8080"}, false)
	if !matches(v6, "http://[::1]:8080") || matches(v6, "http://:8080") || matches(v6, "http://1:8080") {
		t.Errorf("an IPv6 origin %q is not matched exactly", v6)
	}
	star := originPatterns([]string{"https://*.example"}, false)
	if matches(star, "https://evil.example") {
		t.Error("a configured origin was used as a wildcard")
	}
}
