package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

// This file holds the security headers every response carries, and the
// content security policy (CSP) of the HTML documents.
//
// The CSP is computed, never maintained by hand. Every response gets the
// base policy, whose script-src allows only scripts from this origin. An HTML
// document the static handler serves gets the same policy with a
// 'sha256-...' source added to script-src for each of its inline scripts
// (the app shell has two: the theme bootstrap of web/src/app.html and
// SvelteKit's start script). The hashes come from the bytes actually served,
// computed when the file is first served and again whenever it changes (see
// spa.digest), so a rebuilt web client, an edited --static directory and the
// embedded build all carry the right policy without a list to keep in step.
//
// A development server (Config.Dev) enforces the same policy, except that
// it lets same-origin pages frame the app (frame-ancestors 'self' and
// X-Frame-Options SAMEORIGIN), because the test build's /dev/gallery shows
// game fixtures in iframes; a production server forbids framing
// altogether. The policy is enforced, not report-only, in development too,
// for two reasons. The end-to-end tests run a --dev server, and only an
// enforced policy makes them prove the app works under it: Chrome logs a
// report-only violation at the console's info level, which the tests'
// console-error check never sees. And with the Vite dev server, which
// serves the pages itself and forwards only /ws, /api and /health here,
// this server's policy never reaches a page, so it cannot get in the way
// of a dev setup.
//
// HSTS is deliberately not sent from here: one deployment serves plain HTTP
// on purpose, so Strict-Transport-Security belongs to the TLS-terminating
// proxy of each deployment.

// permissionsPolicy turns off the powerful features the app never uses, so
// that an injected script or a framed page could not use them either. It
// names only features Chrome recognises (an unknown one makes it log an
// error in the console), and none the app uses: the app plays sound through
// Web Audio and vibrates through navigator.vibrate, neither of which has a
// Permissions-Policy feature, copies device links to the clipboard
// (clipboard-write stays allowed) and may go full screen (fullscreen stays
// allowed).
const permissionsPolicy = "accelerometer=(), bluetooth=(), camera=(), display-capture=(), " +
	"geolocation=(), gyroscope=(), hid=(), idle-detection=(), magnetometer=(), " +
	"microphone=(), midi=(), payment=(), serial=(), usb=(), xr-spatial-tracking=()"

// headerCSP is the header the policy goes in.
const headerCSP = "Content-Security-Policy"

// securityPolicy decides the security headers of a server.
type securityPolicy struct {
	dev  bool   // allow same-origin framing (the test build's gallery)
	base string // the CSP without inline-script hashes, prebuilt
}

func newSecurityPolicy(dev bool) securityPolicy {
	p := securityPolicy{dev: dev}
	p.base = p.csp(nil)
	return p
}

// csp returns the policy with hashes (the 'sha256-...' sources of the
// document's inline scripts) added to script-src.
//
// style-src allows inline styles because Svelte's style: directives and the
// tooltip placement write style attributes. img-src allows data: and blob:
// URLs for generated images. connect-src 'self' covers the same-origin ws:
// and wss: WebSocket URLs in current browsers (CSP level 3). worker-src and
// manifest-src name the service worker and the web app manifest.
func (p securityPolicy) csp(hashes []string) string {
	frame := "'none'"
	if p.dev {
		frame = "'self'"
	}
	var b strings.Builder
	b.WriteString("default-src 'self'; script-src 'self'")
	for _, h := range hashes {
		b.WriteString(" '")
		b.WriteString(h)
		b.WriteByte('\'')
	}
	b.WriteString("; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; " +
		"connect-src 'self'; worker-src 'self'; manifest-src 'self'; frame-ancestors ")
	b.WriteString(frame)
	b.WriteString("; base-uri 'self'; form-action 'self'; object-src 'none'")
	return b.String()
}

// documentCSP returns the policy of an HTML document: the base policy plus
// the hash of each of its inline scripts.
func (p securityPolicy) documentCSP(doc []byte) string {
	return p.csp(scriptHashes(doc))
}

// setHeaders sets the security headers on h, the CSP being the base one.
func (p securityPolicy) setHeaders(h http.Header) {
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "same-origin")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	h.Set("Permissions-Policy", permissionsPolicy)
	if p.dev {
		h.Set("X-Frame-Options", "SAMEORIGIN")
	} else {
		h.Set("X-Frame-Options", "DENY")
	}
	h.Set(headerCSP, p.base)
}

// scriptHashes returns the CSP hash source ("sha256-<base64>") of each
// distinct inline script of an HTML document, in document order.
func scriptHashes(doc []byte) []string {
	var out []string
	seen := make(map[string]bool)
	for _, s := range inlineScripts(doc) {
		sum := sha256.Sum256(s)
		h := "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	return out
}

// inlineScripts returns the text of every non-empty inline script element of
// an HTML document: the bytes between <script ...> and </script>, with line
// breaks normalized to \n as the HTML parser does before a browser hashes the
// text. A script with a src attribute runs the file it names, not its text,
// so it is skipped, as are scripts inside comments.
//
// It is a small scanner, not an HTML parser (the standard library has none),
// written for the documents this server serves: the web build's HTML, which
// a build tool wrote. Text such as "<script" inside another raw-text element
// (style, textarea, title) would mislead it; the build has none.
func inlineScripts(doc []byte) [][]byte {
	var out [][]byte
	for i := 0; i < len(doc); {
		lt := bytes.IndexByte(doc[i:], '<')
		if lt < 0 {
			break
		}
		i += lt
		rest := doc[i:]
		if bytes.HasPrefix(rest, []byte("<!--")) {
			end := bytes.Index(rest[4:], []byte("-->"))
			if end < 0 {
				break
			}
			i += 4 + end + 3
			continue
		}
		if !hasTagPrefix(rest, "<script") {
			i++
			continue
		}
		tagEnd, hasSrc, ok := scanStartTag(doc, i+len("<script"))
		if !ok {
			break
		}
		body := tagEnd
		end := indexCloseTag(doc[body:], "</script")
		if end < 0 {
			break
		}
		if text := normalizeNewlines(doc[body : body+end]); !hasSrc && len(text) > 0 {
			out = append(out, text)
		}
		i = body + end + len("</script")
		if gt := bytes.IndexByte(doc[i:], '>'); gt >= 0 {
			i += gt + 1
		} else {
			break
		}
	}
	return out
}

// hasTagPrefix reports whether s starts with tag (case-insensitively)
// followed by the end of the tag name: white space, '/' or '>'.
func hasTagPrefix(s []byte, tag string) bool {
	if len(s) <= len(tag) || !strings.EqualFold(string(s[:len(tag)]), tag) {
		return false
	}
	return isTagNameEnd(s[len(tag)])
}

func isTagNameEnd(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\f', '\r', '/', '>':
		return true
	}
	return false
}

func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\f', '\r':
		return true
	}
	return false
}

// scanStartTag reads the attributes of a start tag from doc[i:] (just after
// its name) up to its closing '>'. It returns the index just past the '>',
// whether the tag has a src attribute, and false if the tag never closes.
func scanStartTag(doc []byte, i int) (int, bool, bool) {
	hasSrc := false
	for i < len(doc) {
		c := doc[i]
		switch {
		case c == '>':
			return i + 1, hasSrc, true
		case isSpace(c) || c == '/':
			i++
			continue
		}
		start := i
		for i < len(doc) && !isSpace(doc[i]) && doc[i] != '/' && doc[i] != '>' && doc[i] != '=' {
			i++
		}
		if strings.EqualFold(string(doc[start:i]), "src") {
			hasSrc = true
		}
		for i < len(doc) && isSpace(doc[i]) {
			i++
		}
		if i >= len(doc) || doc[i] != '=' {
			continue
		}
		i++
		for i < len(doc) && isSpace(doc[i]) {
			i++
		}
		if i < len(doc) && (doc[i] == '"' || doc[i] == '\'') {
			q := doc[i]
			end := bytes.IndexByte(doc[i+1:], q)
			if end < 0 {
				return 0, false, false
			}
			i += 1 + end + 1
			continue
		}
		for i < len(doc) && !isSpace(doc[i]) && doc[i] != '>' {
			i++
		}
	}
	return 0, false, false
}

// indexCloseTag returns the index in s of the first end tag named by tag
// ("</script"), matched case-insensitively and followed by the end of the
// name, or -1.
func indexCloseTag(s []byte, tag string) int {
	for k := 0; k < len(s); k++ {
		if s[k] == '<' && hasTagPrefix(s[k:], tag) {
			return k
		}
	}
	return -1
}

// normalizeNewlines turns \r\n and lone \r into \n, as the HTML parser does.
func normalizeNewlines(b []byte) []byte {
	if bytes.IndexByte(b, '\r') < 0 {
		return b
	}
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
}
