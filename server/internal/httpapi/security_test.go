package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// The tests of security.go.

// cspHash is the CSP source that allows an inline script whose text is s.
func cspHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// Every response carries the security headers, whatever answers it: the
// health check, the API, the reserved /ws, the app shell, a static file, an
// immutable asset, versioned art and a 404.
func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	h := newTestHandler(t)
	for _, target := range []string{
		"/health", "/api/x", "/ws", "/", "/play/ABC", "/rules", "/favicon.svg",
		"/_app/version.json", "/_app/immutable/entry/start.Bx81.js",
		"/art/cards/baths.avif?v=" + artVersion(), "/missing.js",
	} {
		rec := do(t, h, http.MethodGet, target, nil)
		hd := rec.Header()
		for name, want := range map[string]string{
			"X-Content-Type-Options":       "nosniff",
			"Referrer-Policy":              "same-origin",
			"X-Frame-Options":              "DENY",
			"Cross-Origin-Opener-Policy":   "same-origin",
			"Cross-Origin-Resource-Policy": "same-origin",
			"Permissions-Policy":           permissionsPolicy,
		} {
			if got := hd.Get(name); got != want {
				t.Errorf("GET %s %s = %q, want %q", target, name, got, want)
			}
		}
		csp := hd.Get(headerCSP)
		if !strings.HasPrefix(csp, "default-src 'self'; script-src 'self'") || !strings.Contains(csp, "frame-ancestors 'none'") {
			t.Errorf("GET %s CSP = %q", target, csp)
		}
		if ro := hd.Get(headerCSPReportOnly); ro != "" {
			t.Errorf("GET %s sends a report-only CSP too: %q", target, ro)
		}
		if hsts := hd.Get("Strict-Transport-Security"); hsts != "" {
			t.Errorf("GET %s sends HSTS %q; that belongs to the proxy", target, hsts)
		}
	}
}

// The base policy, directive by directive.
func TestBaseCSP(t *testing.T) {
	want := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; worker-src 'self'; " +
		"manifest-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; " +
		"object-src 'none'"
	if got := newSecurityPolicy(false).base; got != want {
		t.Errorf("base CSP\n got %q\nwant %q", got, want)
	}
}

// A development server sends the policy report-only, so nothing it serves
// is blocked, and lets same-origin pages frame the app (the test build's
// gallery).
func TestDevCSPIsReportOnly(t *testing.T) {
	h, err := New(Config{Version: "dev", Dev: true}, testFS())
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/health", "/", "/favicon.svg"} {
		rec := do(t, h, http.MethodGet, target, nil)
		hd := rec.Header()
		if got := hd.Get(headerCSP); got != "" {
			t.Errorf("GET %s enforces a CSP in dev: %q", target, got)
		}
		ro := hd.Get(headerCSPReportOnly)
		if !strings.HasPrefix(ro, "default-src 'self'; script-src 'self'") || !strings.Contains(ro, "frame-ancestors 'self'") {
			t.Errorf("GET %s report-only CSP = %q", target, ro)
		}
		if got := hd.Get("X-Frame-Options"); got != "SAMEORIGIN" {
			t.Errorf("GET %s X-Frame-Options = %q, want SAMEORIGIN", target, got)
		}
	}
}

const (
	themeScript = "\n      (function () { document.documentElement.dataset.theme = 'dark'; })();\n    "
	startScript = "\n\t\t\t{ __sveltekit_x = { base: '' }; Promise.all([import('./_app/start.js')]); }\n\t\t"
)

func twoScriptPage(theme, start string) string {
	return "<!doctype html><html><head><script>" + theme + "</script>" +
		`<script type="module" src="/_app/immutable/entry/start.Bx81.js"></script>` +
		"<!-- <script>commented out</script> -->" +
		"</head><body><div><SCRIPT type=\"module\" data-x='a>b'>" + start + "</SCRIPT></div></body></html>"
}

// An HTML document gets the hash of each of its inline scripts, and none for
// a script with src or one inside a comment; a document without inline
// scripts (and every non-HTML response) gets the base policy.
func TestDocumentCSPHashesInlineScripts(t *testing.T) {
	fsys := testFS()
	fsys["index.html"] = &fstest.MapFile{Data: []byte(twoScriptPage(themeScript, startScript))}
	h, err := New(Config{Version: "t"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/", "/index.html", "/play/ABC"} {
		csp := do(t, h, http.MethodGet, target, nil).Header().Get(headerCSP)
		want := "script-src 'self' " + cspHash(themeScript) + " " + cspHash(startScript) + ";"
		if !strings.Contains(csp, want) {
			t.Errorf("GET %s CSP = %q, want it to contain %q", target, csp, want)
		}
		if strings.Count(csp, "'sha256-") != 2 {
			t.Errorf("GET %s CSP has %d hashes, want 2: %q", target, strings.Count(csp, "'sha256-"), csp)
		}
	}
	// HEAD and a 304 carry the same policy as the GET.
	first := do(t, h, http.MethodGet, "/", nil)
	if got := do(t, h, http.MethodHead, "/", nil).Header().Get(headerCSP); got != first.Header().Get(headerCSP) {
		t.Errorf("HEAD / CSP = %q, want the GET's", got)
	}
	again := do(t, h, http.MethodGet, "/", http.Header{"If-None-Match": {first.Header().Get("ETag")}})
	if again.Code != http.StatusNotModified || again.Header().Get(headerCSP) != first.Header().Get(headerCSP) {
		t.Errorf("revalidated / = %d with CSP %q, want 304 with the GET's", again.Code, again.Header().Get(headerCSP))
	}
	base := newSecurityPolicy(false).base
	for _, target := range []string{"/rules", "/favicon.svg", "/health"} {
		if got := do(t, h, http.MethodGet, target, nil).Header().Get(headerCSP); got != base {
			t.Errorf("GET %s CSP = %q, want the base policy", target, got)
		}
	}
}

// In a --static directory the policy follows the file: a rebuilt index.html
// whose inline script changed gets the new hash and loses the old one.
func TestDocumentCSPFollowsTheFile(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.html")
	if err := os.WriteFile(index, []byte(twoScriptPage(themeScript, startScript)), 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := New(Config{}, os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	before := do(t, h, http.MethodGet, "/", nil).Header().Get(headerCSP)
	if !strings.Contains(before, cspHash(startScript)) {
		t.Fatalf("CSP %q lacks the start script's hash", before)
	}

	changed := strings.Replace(startScript, "base: ''", "base: '/x'", 1)
	if err := os.WriteFile(index, []byte(twoScriptPage(themeScript, changed)), 0o600); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(index, later, later); err != nil {
		t.Fatal(err)
	}
	after := do(t, h, http.MethodGet, "/", nil).Header().Get(headerCSP)
	if after == before {
		t.Fatal("the CSP did not change when an inline script did")
	}
	if !strings.Contains(after, cspHash(changed)) || strings.Contains(after, cspHash(startScript)) ||
		!strings.Contains(after, cspHash(themeScript)) {
		t.Errorf("CSP after the rebuild = %q", after)
	}
}

// The dev policy carries the hashes too, report-only.
func TestDevDocumentCSPIsReportOnly(t *testing.T) {
	fsys := testFS()
	fsys["index.html"] = &fstest.MapFile{Data: []byte(twoScriptPage(themeScript, startScript))}
	h, err := New(Config{Dev: true}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	hd := do(t, h, http.MethodGet, "/", nil).Header()
	if hd.Get(headerCSP) != "" || !strings.Contains(hd.Get(headerCSPReportOnly), cspHash(themeScript)) {
		t.Errorf("dev / CSP = %q, report-only = %q", hd.Get(headerCSP), hd.Get(headerCSPReportOnly))
	}
}

func TestInlineScripts(t *testing.T) {
	for _, c := range []struct {
		name, doc string
		want      []string
	}{
		{"none", "<!doctype html><p>hi</p>", nil},
		{"one", "<script>a()</script>", []string{"a()"}},
		{"upper case and attributes", `<SCRIPT type="module" nonce=x>b()</ScRiPt >`, []string{"b()"}},
		{"src is skipped", `<script src="/x.js"></script><script defer SRC='/y.js'>ignored</script>`, nil},
		{"data-src is not src", `<script data-src="x">c()</script>`, []string{"c()"}},
		{"quoted >", `<script data-x="a>b" data-y='<script>'>d()</script>`, []string{"d()"}},
		{"empty is skipped", "<script></script><script>e()</script>", []string{"e()"}},
		{"comments are skipped", "<!-- <script>no()</script> --><script>f()</script>", []string{"f()"}},
		{"not a script tag", "<scripts>no</scripts><scriptx>no</scriptx><script>g()</script>", []string{"g()"}},
		{"close tag inside a string ends it", `<script>h("</script>")</script>`, []string{`h("`}},
		{"line breaks normalized", "<script>\r\ni()\rj()\n</script>", []string{"\ni()\nj()\n"}},
		{"self-closing", "<script/>k()</script>", []string{"k()"}},
		{"unclosed tag", "<script type=x", nil},
		{"unclosed element", "<script>l()", nil},
	} {
		var got []string
		for _, s := range inlineScripts([]byte(c.doc)) {
			got = append(got, string(s))
		}
		if strings.Join(got, "|") != strings.Join(c.want, "|") || len(got) != len(c.want) {
			t.Errorf("%s: inlineScripts = %q, want %q", c.name, got, c.want)
		}
	}
}

// The same script twice is hashed once.
func TestScriptHashesDeduplicate(t *testing.T) {
	got := scriptHashes([]byte("<script>a()</script><script>b()</script><script>a()</script>"))
	want := []string{strings.Trim(cspHash("a()"), "'"), strings.Trim(cspHash("b()"), "'")}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("scriptHashes = %q, want %q", got, want)
	}
}

// The Permissions-Policy turns off features the app never uses and none it
// does (vibration and Web Audio have no feature to turn off; the clipboard
// and full screen stay allowed).
func TestPermissionsPolicyLeavesWhatTheAppUses(t *testing.T) {
	for _, off := range []string{"camera=()", "microphone=()", "geolocation=()", "payment=()", "usb=()"} {
		if !strings.Contains(permissionsPolicy, off) {
			t.Errorf("Permissions-Policy lacks %s", off)
		}
	}
	for _, used := range []string{"vibrate", "clipboard", "fullscreen", "autoplay"} {
		if strings.Contains(permissionsPolicy, used) {
			t.Errorf("Permissions-Policy names %s, which the app uses", used)
		}
	}
}
